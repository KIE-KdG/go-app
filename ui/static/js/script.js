function chatApp() {
  return {
    ws: null,
    inputMessage: '',
    messages: [],
    init() {
      // Establish a WebSocket connection to ws://localhost:4000
      this.ws = new WebSocket('wss://localhost:4000/ws');
      this.ws.addEventListener('open', () => {
        console.log('WebSocket connection established');
      });
      this.ws.addEventListener('message', (event) => {
        console.log('Received:', event.data);
        let data;
        try {
          data = JSON.parse(event.data);
        } catch (e) {
          data = { response: event.data };
        }
        this.messages.push({
          sender: 'AI',
          text: data.response || event.data
        });
        // Ensure the chat messages container scrolls to the bottom
        this.$nextTick(() => {
          const chatMessages = document.getElementById('chatMessages');
          chatMessages.scrollTop = chatMessages.scrollHeight;
        });
      });
      this.ws.addEventListener('close', () => {
        console.log('WebSocket connection closed');
      });
      this.ws.addEventListener('error', (error) => {
        console.error('WebSocket error:', error);
      });
    },
    sendMessage() {
      if (!this.inputMessage.trim()) return;
      // Add user's message to the chat
      this.messages.push({
        sender: 'You',
        text: this.inputMessage
      });
      // Send the message to the server via WebSocket in JSON format
      if (this.ws && this.ws.readyState === WebSocket.OPEN) {
        this.ws.send(JSON.stringify({ message: this.inputMessage }));
      } else {
        console.error('WebSocket is not open');
      }
      // Clear the input field
      this.inputMessage = '';
    }
  }
}