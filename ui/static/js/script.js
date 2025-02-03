(function() {
  const chatContainer = document.getElementById('chat-container');
  const chatForm = document.getElementById('chat-form');
  const messageInput = document.getElementById('message-input');

  // Handle the form submission.
  chatForm.addEventListener('submit', function(event) {
    event.preventDefault();

    const message = messageInput.value.trim();
    if (!message) return;

    // Append the user's message to the chat container.
    appendMessage('user', message);

    // Simulate a delay before the LLM response.
    setTimeout(() => {
      // Hardcoded LLM response.
      appendMessage('bot', 'I am on holiday. shush.');
    }, 500); // Adjust the delay (in ms) as desired.

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
