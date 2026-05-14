// Defer-rel töltöttem be a scriptet, így a html elemek már léteznek, amikor ezt a kódot futtatom
const form = document.getElementById("detectionForm");

async function imageUpload(image, description) {
    const formData = new FormData();

    // Egyedi azonosító generálása
    const jobID = crypto.randomUUID();
    formData.append('jobID', jobID);
    formData.append('image', image);
    formData.append('description', description)
    const response = await fetch('/process', {
        method: 'POST',
        body: formData
    });

    if(response.ok){
        console.log('Sikeres képfeltöltés!');
        return jobID
    } else {
        console.error('Hiba történt a kép processzálása során!');
        const text = await response.text();
        throw  new Error(`Hálózati hiba: ${response.status} - ${text}`);
    }
}

form.addEventListener("submit", async (event) => {
    event.preventDefault();
    const formElem = event.target;

    const image = formElem.image.files[0];
    if(!image){
        console.error("Nincs kiválasztott kép!");
        return;
    }
    const description = formElem.description.value;
    if(!description){
        console.error("Nincs megadva leírás!");
        return;
    }

    try {
        const jobID = await imageUpload(image, description);

        const socket = new WebSocket(
            `ws://${window.location.host}/ws?jobID=${jobID}`
        );

        socket.onopen = () => {
            console.log("Websocket kapcsolat létrejött")
        }

        socket.onmessage = (event) => {
            console.log("Websocket üzenet érkezett");
            const result = JSON.parse(event.data);

            //Visszaérkezett eredmény megjelenítése
            let imagecontainer = document.getElementById("imageContainer");
            let resultcontainer = document.getElementById("resultContainer");
            imagecontainer.innerHTML = '';
            if (result.image){
                //Ha érkezett vissza kép, akkor megjelenítem
                imagecontainer.parentNode.style.display="flex";
                const img = document.createElement("img");
                img.src = "data:image/png;base64," + result.image;
                img.alt = "Feldolgozott kép";
                img.style.maxWidth = "100%";
                img.style.height = "auto";
                imagecontainer.appendChild(img);
            }else {
                imagecontainer.textContent = "Nem érkezett vissza kép az OCR feldolgozásból.";
            }
            if (result.results){
                //Ha érkezett vissza szöveg, akkor megjelenítem
                let pre = resultcontainer.querySelector("pre");
                if (!pre) {
                    pre = document.createElement("pre");
                    resultcontainer.appendChild(pre);
                }
                pre.textContent = result.results.join("\n");
            }

            //Websocket lezárása
            socket.close();
        };

        socket.onclose = () => {
            console.log("WebSocket kapcsolat lezárva");
        };

        socket.onerror = (error) => {
            console.error("WebSocket hiba:", error);
        };

    } catch (error) {
        console.error("Hiba a kép kódolásakor:", error);
        return;
    }
});