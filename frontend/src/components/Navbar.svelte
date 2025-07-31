<script>
  export let logo = '/icons/cloud.svg';
</script>

<nav class="navbar">
  <div class="nav-left">
    <img class="logo" src={logo} alt="Logo" />
    <span class="title">NetGraph</span>
  </div>
  <div class="nav-right">
    <img class="avatar" src="https://placekitten.com/40/40" alt="Avatar" />
  </div>
</nav>

<style>
.navbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 1rem;
  height: 50px;
  background: white;
  position: relative;
  border-bottom: 1px solid #e5e7eb;
}

.navbar::after {
  content: '';
  position: absolute;
  left: 0;
  right: 0;
  bottom: -1px;
  height: 6px;
  border-bottom-left-radius: 8px;
  border-bottom-right-radius: 8px;
  background: linear-gradient(to right, #3b82f6, #9333ea);
}

.nav-left {
  display: flex;
  align-items: center;
}

.logo {
  width: 24px;
  height: 24px;
  margin-right: 0.5rem;
}

.title {
  font-weight: bold;
  font-size: 1.2rem;
}

.avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
}
</style>

