


document.addEventListener("DOMContentLoaded", ()=>{
    const app = document.querySelector("#app");
    setInterval(()=>{
        app.textContent = "DUNG BIETTT"
        setTimeout(()=>{
            app.textContent = ""
        }, 500)
    }, 1000)
})