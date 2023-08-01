<script setup>
const count = ref(0)
const data = ref({})

onMounted(async () => {
    const res = await fetch("/api/peers")
    data.value = await res.json()
})
</script>

<template>
    <h1>home</h1>
    {{ data }}
</template>