
let socket = new WebSocket("ws://localhost:4000/ws");

socket.onopen = function(event) {
    document.getElementById("messages").textContent += "Connected to WebSocket server\n";
};

socket.onmessage = function(event) {
    document.getElementById("messages").textContent += "Received: " + event.data + "\n";
};

socket.onclose = function(event) {
    document.getElementById("messages").textContent += "Disconnected from WebSocket server\n";
};

function sendMessage() {
    let message = document.getElementById("messageInput").value;
    socket.send(message);
    document.getElementById("messageInput").value = "";
}