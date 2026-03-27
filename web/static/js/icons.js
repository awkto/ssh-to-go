// 50 computer-related icons (Feather-style, 24x24 viewBox, stroke-based)
// Each icon: { name, label, svg } where svg is the inner SVG content
(function () {
    "use strict";

    const ICONS = [
        // Favorites
        { name: "terminal", label: "Terminal", svg: '<polyline points="4 17 10 11 4 5"></polyline><line x1="12" y1="19" x2="20" y2="19"></line>' },
        { name: "code", label: "Code", svg: '<polyline points="16 18 22 12 16 6"></polyline><polyline points="8 6 2 12 8 18"></polyline>' },
        { name: "git-commit", label: "Git Commit", svg: '<circle cx="12" cy="12" r="4"></circle><line x1="1.05" y1="12" x2="7" y2="12"></line><line x1="17.01" y1="12" x2="22.96" y2="12"></line>' },
        { name: "git-branch", label: "Git Branch", svg: '<line x1="6" y1="3" x2="6" y2="15"></line><circle cx="18" cy="6" r="3"></circle><circle cx="6" cy="18" r="3"></circle><path d="M18 9a9 9 0 0 1-9 9"></path>' },
        { name: "cloud", label: "Cloud", svg: '<path d="M18 10h-1.26A8 8 0 1 0 9 20h9a5 5 0 0 0 0-10z"></path>' },
        { name: "hash", label: "Hash", svg: '<line x1="4" y1="9" x2="20" y2="9"></line><line x1="4" y1="15" x2="20" y2="15"></line><line x1="10" y1="3" x2="8" y2="21"></line><line x1="16" y1="3" x2="14" y2="21"></line>' },
        { name: "at-sign", label: "At Sign", svg: '<circle cx="12" cy="12" r="4"></circle><path d="M16 8v5a3 3 0 0 0 6 0v-1a10 10 0 1 0-3.92 7.94"></path>' },
        { name: "heart", label: "Heart", svg: '<path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"></path>' },
        { name: "star", label: "Star", svg: '<polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"></polygon>' },
        // General symbols
        { name: "home", label: "Home", svg: '<path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"></path><polyline points="9 22 9 12 15 12 15 22"></polyline>' },
        { name: "bookmark", label: "Bookmark", svg: '<path d="M19 21l-7-5-7 5V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2z"></path>' },
        { name: "flag", label: "Flag", svg: '<path d="M4 15s1-1 4-1 5 2 8 2 4-1 4-1V3s-1 1-4 1-5-2-8-2-4 1-4 1z"></path><line x1="4" y1="22" x2="4" y2="15"></line>' },
        { name: "tag", label: "Tag", svg: '<path d="M20.59 13.41l-7.17 7.17a2 2 0 0 1-2.83 0L2 12V2h10l8.59 8.59a2 2 0 0 1 0 2.82z"></path><line x1="7" y1="7" x2="7.01" y2="7"></line>' },
        { name: "user", label: "User", svg: '<path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"></path><circle cx="12" cy="7" r="4"></circle>' },
        { name: "users", label: "Team", svg: '<path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"></path><circle cx="9" cy="7" r="4"></circle><path d="M23 21v-2a4 4 0 0 0-3-3.87"></path><path d="M16 3.13a4 4 0 0 1 0 7.75"></path>' },
        { name: "briefcase", label: "Work", svg: '<rect x="2" y="7" width="20" height="14" rx="2" ry="2"></rect><path d="M16 21V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v16"></path>' },
        { name: "calendar", label: "Calendar", svg: '<rect x="3" y="4" width="18" height="18" rx="2" ry="2"></rect><line x1="16" y1="2" x2="16" y2="6"></line><line x1="8" y1="2" x2="8" y2="6"></line><line x1="3" y1="10" x2="21" y2="10"></line>' },
        { name: "clock", label: "Clock", svg: '<circle cx="12" cy="12" r="10"></circle><polyline points="12 6 12 12 16 14"></polyline>' },
        { name: "map-pin", label: "Location", svg: '<path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"></path><circle cx="12" cy="10" r="3"></circle>' },
        { name: "compass", label: "Compass", svg: '<circle cx="12" cy="12" r="10"></circle><polygon points="16.24 7.76 14.12 14.12 7.76 16.24 9.88 9.88 16.24 7.76"></polygon>' },
        { name: "sun", label: "Sun", svg: '<circle cx="12" cy="12" r="5"></circle><line x1="12" y1="1" x2="12" y2="3"></line><line x1="12" y1="21" x2="12" y2="23"></line><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"></line><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"></line><line x1="1" y1="12" x2="3" y2="12"></line><line x1="21" y1="12" x2="23" y2="12"></line><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"></line><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"></line>' },
        { name: "moon", label: "Moon", svg: '<path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"></path>' },
        { name: "coffee", label: "Coffee", svg: '<path d="M18 8h1a4 4 0 0 1 0 8h-1"></path><path d="M2 8h16v9a4 4 0 0 1-4 4H6a4 4 0 0 1-4-4V8z"></path><line x1="6" y1="1" x2="6" y2="4"></line><line x1="10" y1="1" x2="10" y2="4"></line><line x1="14" y1="1" x2="14" y2="4"></line>' },
        { name: "anchor", label: "Anchor", svg: '<circle cx="12" cy="5" r="3"></circle><line x1="12" y1="22" x2="12" y2="8"></line><path d="M5 12H2a10 10 0 0 0 20 0h-3"></path>' },
        { name: "truck", label: "Truck", svg: '<rect x="1" y="3" width="15" height="13"></rect><polygon points="16 8 20 8 23 11 23 16 16 16 16 8"></polygon><circle cx="5.5" cy="18.5" r="2.5"></circle><circle cx="18.5" cy="18.5" r="2.5"></circle>' },
        { name: "package", label: "Package", svg: '<line x1="16.5" y1="9.4" x2="7.5" y2="4.21"></line><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"></path><polyline points="3.27 6.96 12 12.01 20.73 6.96"></polyline><line x1="12" y1="22.08" x2="12" y2="12"></line>' },
        { name: "crosshair", label: "Target", svg: '<circle cx="12" cy="12" r="10"></circle><line x1="22" y1="12" x2="18" y2="12"></line><line x1="6" y1="12" x2="2" y2="12"></line><line x1="12" y1="6" x2="12" y2="2"></line><line x1="12" y1="22" x2="12" y2="18"></line>' },
        { name: "award", label: "Award", svg: '<circle cx="12" cy="8" r="7"></circle><polyline points="8.21 13.89 7 23 12 20 17 23 15.79 13.88"></polyline>' },
        { name: "feather", label: "Feather", svg: '<path d="M20.24 12.24a6 6 0 0 0-8.49-8.49L5 10.5V19h8.5z"></path><line x1="16" y1="8" x2="2" y2="22"></line><line x1="17.5" y1="15" x2="9" y2="15"></line>' },
        { name: "hexagon", label: "Hexagon", svg: '<path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"></path>' },
        { name: "droplet", label: "Droplet", svg: '<path d="M12 2.69l5.66 5.66a8 8 0 1 1-11.31 0z"></path>' },
        { name: "scissors", label: "Scissors", svg: '<circle cx="6" cy="6" r="3"></circle><circle cx="6" cy="18" r="3"></circle><line x1="20" y1="4" x2="8.12" y2="15.88"></line><line x1="14.47" y1="14.48" x2="20" y2="20"></line><line x1="8.12" y1="8.12" x2="12" y2="12"></line>' },
        { name: "paperclip", label: "Paperclip", svg: '<path d="M21.44 11.05l-9.19 9.19a6 6 0 0 1-8.49-8.49l9.19-9.19a4 4 0 0 1 5.66 5.66l-9.2 9.19a2 2 0 0 1-2.83-2.83l8.49-8.48"></path>' },
        // Computer & tech icons
        { name: "server", label: "Server", svg: '<rect x="2" y="2" width="20" height="8" rx="2" ry="2"></rect><rect x="2" y="14" width="20" height="8" rx="2" ry="2"></rect><line x1="6" y1="6" x2="6.01" y2="6"></line><line x1="6" y1="18" x2="6.01" y2="18"></line>' },
        { name: "database", label: "Database", svg: '<ellipse cx="12" cy="5" rx="9" ry="3"></ellipse><path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3"></path><path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5"></path>' },
        { name: "monitor", label: "Monitor", svg: '<rect x="2" y="3" width="20" height="14" rx="2" ry="2"></rect><line x1="8" y1="21" x2="16" y2="21"></line><line x1="12" y1="17" x2="12" y2="21"></line>' },
        { name: "laptop", label: "Laptop", svg: '<rect x="3" y="4" width="18" height="12" rx="2" ry="2"></rect><line x1="2" y1="20" x2="22" y2="20"></line>' },
        { name: "smartphone", label: "Phone", svg: '<rect x="5" y="2" width="14" height="20" rx="2" ry="2"></rect><line x1="12" y1="18" x2="12.01" y2="18"></line>' },
        { name: "tablet", label: "Tablet", svg: '<rect x="4" y="2" width="16" height="20" rx="2" ry="2"></rect><line x1="12" y1="18" x2="12.01" y2="18"></line>' },
        { name: "cpu", label: "CPU", svg: '<rect x="4" y="4" width="16" height="16" rx="2" ry="2"></rect><rect x="9" y="9" width="6" height="6"></rect><line x1="9" y1="1" x2="9" y2="4"></line><line x1="15" y1="1" x2="15" y2="4"></line><line x1="9" y1="20" x2="9" y2="23"></line><line x1="15" y1="20" x2="15" y2="23"></line><line x1="20" y1="9" x2="23" y2="9"></line><line x1="20" y1="14" x2="23" y2="14"></line><line x1="1" y1="9" x2="4" y2="9"></line><line x1="1" y1="14" x2="4" y2="14"></line>' },
        { name: "hard-drive", label: "Hard Drive", svg: '<line x1="22" y1="12" x2="2" y2="12"></line><path d="M5.45 5.11L2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11z"></path><line x1="6" y1="16" x2="6.01" y2="16"></line><line x1="10" y1="16" x2="10.01" y2="16"></line>' },
        { name: "wifi", label: "WiFi", svg: '<path d="M5 12.55a11 11 0 0 1 14.08 0"></path><path d="M1.42 9a16 16 0 0 1 21.16 0"></path><path d="M8.53 16.11a6 6 0 0 1 6.95 0"></path><line x1="12" y1="20" x2="12.01" y2="20"></line>' },
        { name: "globe", label: "Globe", svg: '<circle cx="12" cy="12" r="10"></circle><line x1="2" y1="12" x2="22" y2="12"></line><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"></path>' },
        { name: "lock", label: "Lock", svg: '<rect x="3" y="11" width="18" height="11" rx="2" ry="2"></rect><path d="M7 11V7a5 5 0 0 1 10 0v4"></path>' },
        { name: "unlock", label: "Unlock", svg: '<rect x="3" y="11" width="18" height="11" rx="2" ry="2"></rect><path d="M7 11V7a5 5 0 0 1 9.9-1"></path>' },
        { name: "shield", label: "Shield", svg: '<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"></path>' },
        { name: "key", label: "Key", svg: '<path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4"></path>' },
        { name: "folder", label: "Folder", svg: '<path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path>' },
        { name: "file", label: "File", svg: '<path d="M13 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9z"></path><polyline points="13 2 13 9 20 9"></polyline>' },
        { name: "file-text", label: "File Text", svg: '<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path><polyline points="14 2 14 8 20 8"></polyline><line x1="16" y1="13" x2="8" y2="13"></line><line x1="16" y1="17" x2="8" y2="17"></line><polyline points="10 9 9 9 8 9"></polyline>' },
        { name: "bug", label: "Bug", svg: '<rect x="8" y="6" width="8" height="14" rx="4"></rect><path d="M8 10H4"></path><path d="M20 10h-4"></path><path d="M8 18H3"></path><path d="M21 18h-5"></path><path d="M10 6V3"></path><path d="M14 6V3"></path>' },
        { name: "zap", label: "Lightning", svg: '<polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"></polygon>' },
        { name: "activity", label: "Activity", svg: '<polyline points="22 12 18 12 15 21 9 3 6 12 2 12"></polyline>' },
        { name: "layers", label: "Layers", svg: '<polygon points="12 2 2 7 12 12 22 7 12 2"></polygon><polyline points="2 17 12 22 22 17"></polyline><polyline points="2 12 12 17 22 12"></polyline>' },
        { name: "box", label: "Container", svg: '<path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"></path><polyline points="3.27 6.96 12 12.01 20.73 6.96"></polyline><line x1="12" y1="22.08" x2="12" y2="12"></line>' },
        { name: "settings", label: "Settings", svg: '<circle cx="12" cy="12" r="3"></circle><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path>' },
        { name: "tool", label: "Wrench", svg: '<path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z"></path>' },
        { name: "command", label: "Command", svg: '<path d="M18 3a3 3 0 0 0-3 3v12a3 3 0 0 0 3 3 3 3 0 0 0 3-3 3 3 0 0 0-3-3H6a3 3 0 0 0-3 3 3 3 0 0 0 3 3 3 3 0 0 0 3-3V6a3 3 0 0 0-3-3 3 3 0 0 0-3 3 3 3 0 0 0 3 3h12a3 3 0 0 0 3-3 3 3 0 0 0-3-3z"></path>' },
        { name: "link", label: "Link", svg: '<path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"></path><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"></path>' },
        { name: "eye", label: "Eye", svg: '<path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"></path><circle cx="12" cy="12" r="3"></circle>' },
        { name: "bell", label: "Bell", svg: '<path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"></path><path d="M13.73 21a2 2 0 0 1-3.46 0"></path>' },
        { name: "send", label: "Send", svg: '<line x1="22" y1="2" x2="11" y2="13"></line><polygon points="22 2 15 22 11 13 2 9 22 2"></polygon>' },
        { name: "download", label: "Download", svg: '<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path><polyline points="7 10 12 15 17 10"></polyline><line x1="12" y1="15" x2="12" y2="3"></line>' },
        { name: "upload", label: "Upload", svg: '<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path><polyline points="17 8 12 3 7 8"></polyline><line x1="12" y1="3" x2="12" y2="15"></line>' },
        { name: "refresh", label: "Refresh", svg: '<polyline points="23 4 23 10 17 10"></polyline><polyline points="1 20 1 14 7 14"></polyline><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"></path>' },
        { name: "power", label: "Power", svg: '<path d="M18.36 6.64a9 9 0 1 1-12.73 0"></path><line x1="12" y1="2" x2="12" y2="12"></line>' },
        { name: "bluetooth", label: "Bluetooth", svg: '<polyline points="6.5 6.5 17.5 17.5 12 23 12 1 17.5 6.5 6.5 17.5"></polyline>' },
        { name: "radio", label: "Radio", svg: '<circle cx="12" cy="12" r="2"></circle><path d="M16.24 7.76a6 6 0 0 1 0 8.49m-8.48-.01a6 6 0 0 1 0-8.49m11.31-2.82a10 10 0 0 1 0 14.14m-14.14 0a10 10 0 0 1 0-14.14"></path>' },
        { name: "router", label: "Router", svg: '<rect x="2" y="14" width="20" height="8" rx="2"></rect><line x1="6" y1="18" x2="6.01" y2="18"></line><line x1="10" y1="18" x2="10.01" y2="18"></line><path d="M12 2l4 6H8l4-6z"></path>' },
        { name: "gamepad", label: "Gamepad", svg: '<rect x="2" y="6" width="20" height="12" rx="2"></rect><line x1="6" y1="12" x2="10" y2="12"></line><line x1="8" y1="10" x2="8" y2="14"></line><line x1="15" y1="11" x2="15.01" y2="11"></line><line x1="18" y1="13" x2="18.01" y2="13"></line>' },
        { name: "music", label: "Music", svg: '<path d="M9 18V5l12-2v13"></path><circle cx="6" cy="18" r="3"></circle><circle cx="18" cy="16" r="3"></circle>' },
        { name: "camera", label: "Camera", svg: '<path d="M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z"></path><circle cx="12" cy="13" r="4"></circle>' },
    ];

    const ICON_COLORS = [
        { name: "default", label: "Default", hex: "#3b82f6" },
        { name: "green",   label: "Green",   hex: "#22c55e" },
        { name: "red",     label: "Red",     hex: "#ef4444" },
        { name: "orange",  label: "Orange",  hex: "#f59e0b" },
        { name: "purple",  label: "Purple",  hex: "#a78bfa" },
        { name: "pink",    label: "Pink",    hex: "#ec4899" },
        { name: "cyan",    label: "Cyan",    hex: "#06b6d4" },
        { name: "gray",    label: "Gray",    hex: "#94a3b8" },
    ];

    const DEFAULT_ICON = "terminal";
    const DEFAULT_COLOR = "default";

    function colorHex(colorName) {
        var c = ICON_COLORS.find(function(c) { return c.name === colorName; });
        return c ? c.hex : ICON_COLORS[0].hex;
    }

    // Render an icon SVG by name, with optional color name
    function renderIcon(name, size, color) {
        size = size || 16;
        var icon = ICONS.find(function(i) { return i.name === name; });
        if (!icon) return renderIcon(DEFAULT_ICON, size, color);
        var stroke = color ? colorHex(color) : "currentColor";
        return '<svg width="' + size + '" height="' + size + '" viewBox="0 0 24 24" fill="none" stroke="' + stroke + '" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">' + icon.svg + '</svg>';
    }

    // In-memory cache of session icons, loaded from server API.
    // Key format: "host:session" → { icon, color }
    var _sessionIconCache = {};
    var _sessionIconsLoaded = false;

    function loadSessionIcons(callback) {
        if (_sessionIconsLoaded) { if (callback) callback(); return; }
        fetch("/api/session-icons").then(function(r) { return r.json(); }).then(function(data) {
            _sessionIconCache = data || {};
            _sessionIconsLoaded = true;
            if (callback) callback();
        }).catch(function() {
            _sessionIconsLoaded = true;
            if (callback) callback();
        });
    }

    function getSessionIcon(hostName, sessionName) {
        var entry = _sessionIconCache[hostName + ":" + sessionName];
        return entry ? (entry.icon || null) : null;
    }

    function getSessionIconColor(hostName, sessionName) {
        var entry = _sessionIconCache[hostName + ":" + sessionName];
        return entry ? (entry.color || null) : null;
    }

    function setSessionIcon(hostName, sessionName, iconName) {
        var key = hostName + ":" + sessionName;
        var entry = _sessionIconCache[key] || {};
        entry.icon = iconName || "";
        _sessionIconCache[key] = entry;
        _syncSessionIcon(hostName, sessionName, entry);
    }

    function setSessionIconColor(hostName, sessionName, colorName) {
        var key = hostName + ":" + sessionName;
        var entry = _sessionIconCache[key] || {};
        entry.color = colorName || "";
        _sessionIconCache[key] = entry;
        _syncSessionIcon(hostName, sessionName, entry);
    }

    function _syncSessionIcon(hostName, sessionName, entry) {
        fetch("/api/session-icons/" + encodeURIComponent(hostName) + "/" + encodeURIComponent(sessionName), {
            method: "PUT",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ icon: entry.icon || "", color: entry.color || "" }),
        }).catch(function() {});
    }

    // Show icon picker popup attached to an element
    // onSelect(iconName, colorName) is called when an icon is chosen
    // currentIcon/currentColor are the currently selected values
    function showIconPicker(anchorEl, currentIcon, onSelect, currentColor) {
        closeIconPicker();

        currentColor = currentColor || DEFAULT_COLOR;

        var selectedColor = currentColor;
        var selectedIcon = currentIcon || DEFAULT_ICON;
        var picker = document.createElement("div");
        picker.className = "icon-picker-popup";
        picker.id = "icon-picker-popup";

        // Color row
        var colorRow = document.createElement("div");
        colorRow.className = "icon-picker-colors";
        function renderColors() {
            colorRow.innerHTML = ICON_COLORS.map(function(c) {
                var sel = c.name === selectedColor ? " selected" : "";
                return '<button type="button" class="icon-color-swatch' + sel + '" data-color="' + c.name + '" title="' + c.label + '" style="background:' + c.hex + '"></button>';
            }).join("");
        }
        renderColors();

        var search = document.createElement("input");
        search.type = "text";
        search.placeholder = "Search icons...";
        search.className = "icon-picker-search";

        var grid = document.createElement("div");
        grid.className = "icon-picker-grid";

        function renderGrid(filter) {
            var items = ICONS;
            if (filter) {
                var f = filter.toLowerCase();
                items = ICONS.filter(function (i) {
                    return i.name.includes(f) || i.label.toLowerCase().includes(f);
                });
            }
            grid.innerHTML = items.map(function (icon) {
                var sel = icon.name === selectedIcon ? " selected" : "";
                return '<button type="button" class="icon-picker-item' + sel + '" data-icon="' + icon.name + '" title="' + icon.label + '">' +
                    renderIcon(icon.name, 20, selectedColor) +
                    '</button>';
            }).join("");
        }

        renderGrid("");

        search.addEventListener("input", function () {
            renderGrid(search.value);
        });

        grid.addEventListener("click", function (e) {
            e.stopPropagation();
            var btn = e.target.closest(".icon-picker-item");
            if (!btn) return;
            selectedIcon = btn.dataset.icon;
            grid.querySelectorAll(".icon-picker-item").forEach(function(item) {
                item.classList.toggle("selected", item.dataset.icon === selectedIcon);
            });
            onSelect(selectedIcon, selectedColor);
        });

        // Color clicks also apply immediately
        colorRow.addEventListener("click", function(e) {
            e.stopPropagation();
            var btn = e.target.closest(".icon-color-swatch");
            if (!btn) return;
            selectedColor = btn.dataset.color;
            colorRow.querySelectorAll(".icon-color-swatch").forEach(function(s) {
                s.classList.toggle("selected", s.dataset.color === selectedColor);
            });
            renderGrid(search.value);
            onSelect(selectedIcon, selectedColor);
        });

        picker.appendChild(colorRow);
        picker.appendChild(search);
        picker.appendChild(grid);

        document.body.appendChild(picker);

        // Position near the anchor, clamped to viewport
        var rect = anchorEl.getBoundingClientRect();
        var pickerW = 280;
        var left = rect.left;
        var top = rect.bottom + 4;

        if (left + pickerW > window.innerWidth) left = window.innerWidth - pickerW - 8;
        if (left < 8) left = 8;

        // If no room below, try above
        var spaceBelow = window.innerHeight - rect.bottom - 8;
        var spaceAbove = rect.top - 8;
        if (spaceBelow < 200 && spaceAbove > spaceBelow) {
            // Place above, limit height to available space
            var maxH = Math.min(380, spaceAbove);
            top = rect.top - maxH - 4;
            picker.style.maxHeight = maxH + "px";
        } else {
            // Place below, limit height to available space
            var maxH = Math.min(380, spaceBelow);
            picker.style.maxHeight = maxH + "px";
        }
        if (top < 4) top = 4;

        picker.style.left = left + "px";
        picker.style.top = top + "px";

        setTimeout(function () {
            document.addEventListener("click", iconPickerOutsideClick);
        }, 10);

        search.focus();
    }

    function iconPickerOutsideClick(e) {
        var picker = document.getElementById("icon-picker-popup");
        if (picker && !picker.contains(e.target)) {
            closeIconPicker();
        }
    }

    function closeIconPicker() {
        var existing = document.getElementById("icon-picker-popup");
        if (existing) existing.remove();
        document.removeEventListener("click", iconPickerOutsideClick);
    }

    // Build an icon selector field for use in modals (add/edit host)
    // Returns HTML string. currentIcon/currentColor are the current values.
    function iconSelectorHTML(fieldId, currentIcon, currentColor) {
        currentIcon = currentIcon || DEFAULT_ICON;
        currentColor = currentColor || DEFAULT_COLOR;
        var hex = colorHex(currentColor);
        return '<label>Icon</label>' +
            '<div class="icon-selector" id="' + fieldId + '-wrap">' +
                '<button type="button" class="icon-selector-btn" id="' + fieldId + '-btn" data-icon="' + currentIcon + '" data-color="' + currentColor + '">' +
                    '<span class="icon-selector-swatch" style="background:' + hex + '"></span>' +
                    renderIcon(currentIcon, 20, currentColor) +
                    '<span class="icon-selector-label">' + (ICONS.find(function(i) { return i.name === currentIcon; }) || { label: "Terminal" }).label + '</span>' +
                '</button>' +
                '<input type="hidden" id="' + fieldId + '" value="' + currentIcon + '">' +
                '<input type="hidden" id="' + fieldId + '-color" value="' + currentColor + '">' +
            '</div>';
    }

    // Attach icon picker behavior to a selector built with iconSelectorHTML
    function attachIconSelector(fieldId) {
        var btn = document.getElementById(fieldId + "-btn");
        var input = document.getElementById(fieldId);
        var colorInput = document.getElementById(fieldId + "-color");
        if (!btn || !input) return;

        btn.addEventListener("click", function (e) {
            e.preventDefault();
            e.stopPropagation();
            showIconPicker(btn, input.value, function (iconName, colorName) {
                if (!iconName) iconName = DEFAULT_ICON;
                if (!colorName) colorName = DEFAULT_COLOR;
                input.value = iconName;
                if (colorInput) colorInput.value = colorName;
                btn.dataset.icon = iconName;
                btn.dataset.color = colorName;
                var hex = colorHex(colorName);
                btn.innerHTML =
                    '<span class="icon-selector-swatch" style="background:' + hex + '"></span>' +
                    renderIcon(iconName, 20, colorName) +
                    '<span class="icon-selector-label">' + (ICONS.find(function(i) { return i.name === iconName; }) || { label: "Terminal" }).label + '</span>';
            }, colorInput ? colorInput.value : DEFAULT_COLOR);
        });
    }

    // Expose globally
    window.APP_ICONS = ICONS;
    window.ICON_COLORS = ICON_COLORS;
    window.DEFAULT_ICON = DEFAULT_ICON;
    window.DEFAULT_COLOR = DEFAULT_COLOR;
    window.colorHex = colorHex;
    window.loadSessionIcons = loadSessionIcons;
    window.renderIcon = renderIcon;
    window.getSessionIcon = getSessionIcon;
    window.setSessionIcon = setSessionIcon;
    window.getSessionIconColor = getSessionIconColor;
    window.setSessionIconColor = setSessionIconColor;
    window.showIconPicker = showIconPicker;
    window.closeIconPicker = closeIconPicker;
    window.iconSelectorHTML = iconSelectorHTML;
    window.attachIconSelector = attachIconSelector;
})();
