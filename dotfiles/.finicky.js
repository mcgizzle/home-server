module.exports = {
  defaultBrowser: "Firefox Developer Edition",
  handlers: [
    {
      match: finicky.matchHostnames(["meet.google.com"]),
      browser: "Google Chrome"
    },
    {
      match: finicky.matchHostnames(["app.plex.tv"]),
      browser: "Google Chrome"
    },
    {
      match: finicky.matchHostnames(["youtube.com", "www.youtube.com", "youtu.be"]),
      browser: "Firefox"
    }
  ]
};
