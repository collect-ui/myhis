const express = require('express');
const multer = require('multer');
const path = require('path');
const fs = require('fs');

const app = express();
const upload = multer({ dest: 'uploads/' });

app.post('/upload', upload.single('file'), (req, res) => {
  const file = req.file;
  if (!file) {
    return res.status(400).send('No file uploaded.');
  }

  const oldPath = path.join(__dirname, file.path);
  const newPath = path.join(__dirname, 'uploads', file.originalname);

  fs.rename(oldPath, newPath, (err) => {
    if (err) {
      return res.status(500).send('File save failed.');
    }
    res.send('File uploaded successfully.');
  });
});

app.listen(3000, () => {
  console.log('Server is running on port 3000');
});