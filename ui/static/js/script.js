(function() {
  const chatContainer = document.getElementById('chat-container');
  const chatForm = document.getElementById('chat-form');
  const messageInput = document.getElementById('message-input');

  // Handle the form submission.
  chatForm.addEventListener('submit', function(event) {
    event.preventDefault();

    const message = messageInput.value.trim();
    if (!message) return;

    // Append the user's message.
    appendMessage('user', message);

    // Send the prompt to the backend.
    fetch('/api/chat', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ message: message })
    })
    .then(response => {
      if (!response.ok) {
        throw new Error("Network response was not ok");
      }
      return response.json();
    })
    .then(data => {
      // Append the model's response to the chat container.
      appendMessage('bot', data.response);
    })
    .catch(err => {
      console.error('Error:', err);
      appendMessage('bot', 'Sorry, there was an error processing your request.');
    });

    // Clear the input field.
    messageInput.value = '';
  });

  // Helper function to append messages to the chat container.
  function appendMessage(sender, text) {
    const messageElem = document.createElement('div');
    messageElem.classList.add('message', sender);
    messageElem.textContent = text;
    chatContainer.appendChild(messageElem);
    // Automatically scroll to the latest message.
    chatContainer.scrollTop = chatContainer.scrollHeight;
  }
})();
