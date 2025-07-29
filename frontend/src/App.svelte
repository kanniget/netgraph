<script>
import { onMount } from 'svelte';
import * as d3 from 'd3';

const icons = {
    net: '/icons/wifi.svg',
    host: '/icons/computer-desktop.svg',
    rtr: '/icons/server-stack.svg',
    fw: '/icons/shield-check.svg'
};

let graph = {nodes:[], links:[]};

onMount(async () => {
    const res = await fetch('/api/graph');
    graph = await res.json();
    draw();
});

function draw(){
    const svg = d3.select('#graph');
    const width = window.innerWidth;
    const height = window.innerHeight;
    svg.attr('width', width).attr('height', height);

    const simulation = d3.forceSimulation(graph.nodes)
        .force('link', d3.forceLink(graph.links).id(d => d.id).distance(100))
        .force('charge', d3.forceManyBody().strength(-300))
        .force('center', d3.forceCenter(width/2, height/2));

    const link = svg.append('g')
        .attr('stroke', '#999')
        .selectAll('line')
        .data(graph.links)
        .enter().append('line');

    const node = svg.append('g')
        .selectAll('g')
        .data(graph.nodes)
        .enter().append('g')
        .call(d3.drag()
            .on('start', dragstarted)
            .on('drag', dragged)
            .on('end', dragended));

    node.append('image')
        .attr('href', d => icons[d.type])
        .attr('width', 24)
        .attr('height', 24)
        .attr('x', -12)
        .attr('y', -12);

    node.append('text')
        .attr('y', 20)
        .attr('text-anchor', 'middle')
        .text(d => d.name || d.id);

    node.append('title').text(d => d.name || d.id);

    simulation.on('tick', () => {
        link.attr('x1', d => d.source.x)
            .attr('y1', d => d.source.y)
            .attr('x2', d => d.target.x)
            .attr('y2', d => d.target.y);

        node.attr('transform', d => `translate(${d.x},${d.y})`);
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
        event.subject.fx = null;
        event.subject.fy = null;
    }
}
</script>

<main>
    <svg id="graph" style="width:100%; height:100%;"></svg>
</main>

<style>
main,
svg {
    width: 100%;
    height: 100%;
}

svg {
    border: 1px solid #ccc;
}
</style>
