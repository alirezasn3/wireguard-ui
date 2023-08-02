const input = document.querySelector("input");
const button = document.querySelector("button");

button.addEventListener("click", async () => {
  try {
    const res = await fetch("/api/peers", { method: "POST" });
    console.log(res.status);
    const data = await res.json();
    console.log(data);
  } catch (error) {
    console.log(error);
  }
});
