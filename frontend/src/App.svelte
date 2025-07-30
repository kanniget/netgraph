<script>
import { onMount } from 'svelte';
import * as d3 from 'd3';

const icons = {
    net: '/icons/cloud.svg',
    host: '/icons/server-stack.svg',
    vm: '/icons/computer-desktop.svg',
    rtr: '/icons/router-network.svg',
    fw: '/icons/shield-check.svg',
    zone: '/icons/wifi.svg',
    bridge: '/icons/bridge.svg',
    nic: '/icons/nic.svg',
    disk: '/icons/disk.svg'
};

let graph = {nodes:[], links:[]};
let files = [];
let selectedFile = '';
let showWeights = false;
let typeWeights = [];
let showAttract = false;
let typeAttract = [];
let attractMap = {};
let simulation;
let selectedNodes = [];
let pathLinks = [];
let pathColor = '#ff0000';
let linkSelection;
let nodeSelection;
let adjacency = new Map();
let linkMap = new Map();
let locatedNodeId = '';
let highlightedNode = null;
let showHide = false;
let typeHide = [];
let hiddenTypes = new Set(['disk']);
let fixMode = false;
let showInfo = false;
let infoNode = null;

function nodeType(nodeRef){
    if(typeof nodeRef === 'object' && nodeRef !== null){
        return nodeRef.type;
    }
    const found = graph.nodes.find(n => n.id === nodeRef);
    return found ? found.type : '';
}

function openWeightsDialog(){
    const map = {};
    graph.links.forEach(l => {
        const sType = nodeType(l.source);
        const tType = nodeType(l.target);
        const key = sType + '-' + tType;
        if(!map[key]){
            map[key] = {sourceType: sType, targetType: tType, weight: l.weight || 1};
        }
    });
    typeWeights = Object.values(map);
    showWeights = true;
}

function openAttractDialog(){
    const map = {};
    graph.nodes.forEach(n => {
        if(!map[n.type]){
            map[n.type] = {type: n.type, strength: attractMap[n.type] || -300};
        }
    });
    typeAttract = Object.values(map);
    showAttract = true;
}

function openHideDialog(){
    const map = {};
    graph.nodes.forEach(n => {
        if(!map[n.type]){
            map[n.type] = {type: n.type, visible: !hiddenTypes.has(n.type)};
        }
    });
    typeHide = Object.values(map);
    showHide = true;
}

onMount(async () => {
    const fRes = await fetch('/api/files');
    files = await fRes.json();
    selectedFile = files[0] || 'graph.json';
    await loadGraph();
});

async function loadGraph(){
    const svg = d3.select('#graph');
    svg.selectAll('*').remove();
    const res = await fetch(`/api/graph?file=${selectedFile}`);
    graph = await res.json();
    graph.links.forEach(l => {
        if (l.weight === undefined) l.weight = 1;
    });
    graph.nodes.forEach(n => {
        if(attractMap[n.type] === undefined) attractMap[n.type] = -300;
    });
    locatedNodeId = '';
    highlightedNode = null;
    draw();
}

function draw(){
    const svg = d3.select('#graph');
    svg.selectAll('*').remove();
    if (simulation) simulation.stop();
    const width = window.innerWidth;
    const height = window.innerHeight;
    svg.attr('width', width).attr('height', height);

    const container = svg.append('g');

    const zoom = d3.zoom()
        .scaleExtent([0.5, 5])
        .on('zoom', (event) => {
            container.attr('transform', event.transform);
        });

    svg.call(zoom);

    const nodes = graph.nodes.filter(n => !hiddenTypes.has(n.type));
    const nodeIds = new Set(nodes.map(n => n.id));
    const links = graph.links.filter(l => {
        const s = typeof l.source === 'object' ? l.source.id : l.source;
        const t = typeof l.target === 'object' ? l.target.id : l.target;
        return nodeIds.has(s) && nodeIds.has(t);
    });

    buildMaps(nodes, links);

    simulation = d3.forceSimulation(nodes)
        .force('link', d3.forceLink(links).id(d => d.id).distance(l => 200 / (l.weight || 1)))
        .force('charge', d3.forceManyBody().strength(d => attractMap[d.type] || -300))
        .force('center', d3.forceCenter(width/2, height/2));

    linkSelection = container.append('g')
        .attr('stroke', '#999')
        .attr('stroke-width', 1)
        .selectAll('line')
        .data(links)
        .enter().append('line');

    nodeSelection = container.append('g')
        .selectAll('g')
        .data(nodes)
        .enter().append('g')
        .call(d3.drag()
            .on('start', dragstarted)
            .on('drag', dragged)
            .on('end', dragended));
    nodeSelection.on('click', nodeClicked);
    nodeSelection.on('dblclick', nodeDblClicked);

    nodeSelection.append('image')
        .attr('href', d => icons[d.type])
        .attr('width', d => (d.type === 'net' || d.type === 'zone') ? 96 : 24)
        .attr('height', d => (d.type === 'net' || d.type === 'zone') ? 96 : 24)
        .attr('x', d => (d.type === 'net' || d.type === 'zone') ? -48 : -12)
        .attr('y', d => (d.type === 'net' || d.type === 'zone') ? -48 : -12);

    nodeSelection.append('circle')
        .attr('r', d => (d.type === 'net' || d.type === 'zone') ? 48 : 12)
        .attr('fill', 'none')
        .attr('pointer-events', 'none');

    nodeSelection.append('text')
        .attr('y', 20)
        .attr('text-anchor', 'middle')
        .text(d => d.name || d.id);

    nodeSelection.append('text')
        .attr('y', 32)
        .attr('text-anchor', 'middle')
        .attr('font-size', '10px')
        .text(d => d.type);

    nodeSelection.append('title').text(d => d.name || d.id);

    simulation.on('tick', () => {
        linkSelection.attr('x1', d => d.source.x)
            .attr('y1', d => d.source.y)
            .attr('x2', d => d.target.x)
            .attr('y2', d => d.target.y);

        nodeSelection.attr('transform', d => `translate(${d.x},${d.y})`);
    });

    function dragstarted(event) {
        if (!event.active) simulation.alphaTarget(0.3).restart();
        event.subject.fx = event.subject.x;
        event.subject.fy = event.subject.y;
    }

    function dragged(event) {
        event.subject.fx = event.x;
        event.subject.fy = event.y;
    }

    function dragended(event) {
        if (!event.active) simulation.alphaTarget(0);
        if (fixMode) {
            event.subject.fx = event.x;
            event.subject.fy = event.y;
        } else {
            event.subject.fx = null;
            event.subject.fy = null;
        }
    }
}

function buildMaps(nodesList = graph.nodes, linksList = graph.links){
    adjacency = new Map();
    linkMap = new Map();
    linksList.forEach(l => {
        const s = typeof l.source === 'object' ? l.source.id : l.source;
        const t = typeof l.target === 'object' ? l.target.id : l.target;
        if(!adjacency.has(s)) adjacency.set(s, []);
        if(!adjacency.has(t)) adjacency.set(t, []);
        adjacency.get(s).push(t);
        adjacency.get(t).push(s);
        linkMap.set(s + '-' + t, l);
        linkMap.set(t + '-' + s, l);
    });
}

function findPath(startId, endId){
    const queue = [[startId, []]];
    const visited = new Set();
    while(queue.length){
        const [node, path] = queue.shift();
        if(node === endId) return path;
        if(visited.has(node)) continue;
        visited.add(node);
        const neighbors = adjacency.get(node) || [];
        neighbors.forEach(n => {
            if(!visited.has(n)){
                const l = linkMap.get(node + '-' + n);
                queue.push([n, [...path, l]]);
            }
        });
    }
    return [];
}

function updateHighlights(){
    nodeSelection.select('circle')
        .classed('selected', d => selectedNodes.includes(d))
        .classed('located', d => highlightedNode && d.id === highlightedNode.id);
    linkSelection
        .attr('stroke', d => pathLinks.includes(d) ? pathColor : '#999')
        .attr('stroke-width', d => pathLinks.includes(d) ? 3 : 1);
}

function highlightFromList(){
    highlightedNode = graph.nodes.find(n => n.id === locatedNodeId) || null;
    updateHighlights();
}

function nodeClicked(event, d){
    if(selectedNodes.length === 2){
        selectedNodes = [];
        pathLinks = [];
    }
    if(!selectedNodes.includes(d)){
        selectedNodes.push(d);
    }
    if(selectedNodes.length === 2){
        pathLinks = findPath(selectedNodes[0].id, selectedNodes[1].id);
    }
    updateHighlights();
}

function nodeDblClicked(event, d){
    infoNode = d;
    showInfo = true;
}

function applyWeights() {
    graph.links.forEach(l => {
        const sType = nodeType(l.source);
        const tType = nodeType(l.target);
        const match = typeWeights.find(tw => tw.sourceType === sType && tw.targetType === tType);
        if(match){
            l.weight = match.weight;
        }
    });
    simulation.force('link').distance(l => 200 / (l.weight || 1));
    simulation.alpha(1).restart();
    showWeights = false;
}

function applyAttract() {
    typeAttract.forEach(ta => {
        attractMap[ta.type] = ta.strength;
    });
    simulation.force('charge').strength(d => attractMap[d.type] || -300);
    simulation.alpha(1).restart();
    showAttract = false;
}

function applyHide() {
    hiddenTypes = new Set(typeHide.filter(th => !th.visible).map(th => th.type));
    draw();
    showHide = false;
}
</script>

<main>
    <div style="position:absolute;top:10px;left:10px;z-index:10;background:white;padding:4px;border-radius:4px;">
        <select bind:value={selectedFile} on:change={loadGraph}>
            {#each files as f}
                <option value={f}>{f}</option>
            {/each}
        </select>
        <button on:click={openWeightsDialog} style="margin-left:4px;">Weights</button>
        <button on:click={openAttractDialog} style="margin-left:4px;">Attractiveness</button>
        <button on:click={openHideDialog} style="margin-left:4px;">Hide Types</button>
        <label style="margin-left:4px;">
            <input type="checkbox" bind:checked={fixMode}> Fix Mode
        </label>
        <input type="color" bind:value={pathColor} on:input={updateHighlights} style="margin-left:4px;" title="Path color"/>
        <select bind:value={locatedNodeId} on:change={highlightFromList} style="margin-left:4px;">
            <option value="">Find node...</option>
            {#each graph.nodes.filter(n => !hiddenTypes.has(n.type)) as n}
                <option value={n.id}>{n.name || n.id}</option>
            {/each}
        </select>
    </div>
    {#if showWeights}
    <div class="dialog">
        <div class="dialog-content">
            <h3>Link Weights</h3>
            {#each typeWeights as tw}
            <div class="weight-row">
                <span>{tw.sourceType} - {tw.targetType}</span>
                <input type="number" min="0.1" step="0.1" bind:value={tw.weight}>
            </div>
            {/each}
            <div class="buttons">
                <button on:click={applyWeights}>Apply</button>
                <button on:click={() => showWeights = false}>Close</button>
            </div>
        </div>
    </div>
    {/if}
    {#if showAttract}
    <div class="dialog">
        <div class="dialog-content">
            <h3>Node Attractiveness</h3>
            {#each typeAttract as ta}
            <div class="weight-row">
                <span>{ta.type}</span>
                <input type="number" step="10" bind:value={ta.strength}>
            </div>
            {/each}
            <div class="buttons">
                <button on:click={applyAttract}>Apply</button>
                <button on:click={() => showAttract = false}>Close</button>
            </div>
        </div>
    </div>
    {/if}
    {#if showHide}
    <div class="dialog">
        <div class="dialog-content">
            <h3>Hide Node Types</h3>
            {#each typeHide as th}
            <div class="weight-row">
                <span>{th.type}</span>
                <input type="checkbox" bind:checked={th.visible}>
            </div>
            {/each}
            <div class="buttons">
                <button on:click={applyHide}>Apply</button>
                <button on:click={() => showHide = false}>Close</button>
            </div>
        </div>
    </div>
    {/if}
    {#if showInfo}
    <div class="dialog">
        <div class="dialog-content">
            <h3>{infoNode.name || infoNode.id}</h3>
            <pre>{JSON.stringify(infoNode.info || {}, null, 2)}</pre>
            <div class="buttons">
                <button on:click={() => showInfo = false}>Close</button>
            </div>
        </div>
    </div>
    {/if}
    <svg id="graph" style="width:100%; height:100%;"></svg>
</main>

<style>
main,
svg {
    width: 100%;
    height: 100%;
}

main {
    position: relative;
}

svg {
    border: 1px solid #ccc;
}

.dialog {
    position: absolute;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    background: white;
    padding: 10px;
    border: 1px solid #ccc;
    border-radius: 4px;
    z-index: 20;
}

.dialog-content h3 {
    margin-top: 0;
}

.weight-row {
    display: flex;
    justify-content: space-between;
    margin-bottom: 4px;
}

.buttons {
    margin-top: 8px;
    text-align: right;
}

circle.selected {
    stroke: red;
    stroke-width: 2px;
}

circle.located {
    stroke: blue;
    stroke-width: 2px;
}
</style>
