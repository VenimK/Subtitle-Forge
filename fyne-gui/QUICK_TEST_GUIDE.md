# Quick Test Guide - Improved Translation

## 🎯 What's New?

Your translator now uses the **same approach as your better Python translator**:
- ✅ Structured JSON with enforced schema
- ✅ Safety settings disabled (BLOCK_NONE)
- ✅ Detailed instruction prompts
- ✅ Better error handling

## 🚀 How to Test

### Method 1: Use the GUI (Easiest)

1. **Run the application:**
   ```bash
   cd fyne-gui
   ./subtitle-forge
   # or: go run .
   ```

2. **Go to AI Translation tab**

3. **Configure settings:**
   - **API Key**: Your Gemini API key (get from https://aistudio.google.com/app/apikey)
   - **Model**: Keep "gemini-2.5-flash"
   - **Target Language**: Choose (e.g., "Portuguese")
   - **Temperature**: Try 0.3 (lower = more consistent)
   - **Description**: Add context like "Casual conversation between friends"

4. **Select a subtitle file** and click "Start Translation"

5. **Compare results** with your Python translator!

### Method 2: Quick Command-Line Test

Create a simple test file:
```bash
cd fyne-gui

# Create a test subtitle
cat > test.srt << 'EOF'
1
00:00:01,000 --> 00:00:03,000
Hello, how are you?

2
00:00:03,500 --> 00:00:05,000
I'm doing great, thanks!
EOF

# Set your API key
export GEMINI_API_KEY='your-api-key-here'

# Run your app and translate through the GUI
./subtitle-forge
```

## 🔍 What to Look For

### Quality Improvements
- **Better context understanding** with descriptions
- **More consistent translations** (especially with lower temperature)
- **No content blocking** on mature content
- **Exact format preservation** (all entries, timing, structure)

### Settings to Try

#### For Movies/TV Shows
```
Temperature: 0.2-0.3
Description: "Action movie with casual dialogue. Use natural, conversational [target language]."
```

#### For Documentaries
```
Temperature: 0.1-0.2
Description: "Nature documentary. Use formal, descriptive language with proper terminology."
```

#### For Anime/Casual Content
```
Temperature: 0.3-0.5
Description: "Anime series with young characters. Use casual, friendly language."
```

## 📊 Compare with Python Translator

Test the **same subtitle file** with both translators and compare:

### Your Python Translator
```bash
python your_translator.py --input test.srt --output test_python.srt --target Portuguese
```

### Your Go Translator (Now Improved!)
```bash
# Use the GUI or call TestTranslation() function
```

### Look for:
- Translation quality and naturalness
- Handling of formatting and special characters
- Consistency across multiple lines
- Context understanding from description

## 🎨 Advanced: Test Different Contexts

Try these description examples:

```
"Medical drama TV series. Use medical terminology accurately."
"Comedy show with sarcasm and jokes. Keep the humor natural."
"Children's animated movie. Use simple, age-appropriate language."
"Tech tutorial video. Use technical terms but keep explanations clear."
"Romantic drama. Use emotional, expressive language."
```

## 🐛 Troubleshooting

### Translation seems inconsistent?
- **Lower the temperature** to 0.1-0.2
- **Add more context** in description field

### API errors?
- Check your API key is valid
- Verify you have quota remaining at https://aistudio.google.com/
- Wait a moment if you hit rate limits

### Different results than Python version?
- Make sure temperature is the same
- Use the same description/context
- Python version may have additional post-processing

## 💡 Pro Tips

1. **Temperature matters**: 
   - 0.1-0.2 = Very consistent, more literal
   - 0.3-0.5 = Balanced (recommended)
   - 0.6-1.0 = More creative, less consistent

2. **Be specific in descriptions**:
   - ❌ "Translate this"
   - ✅ "Casual conversation in a coffee shop between friends"

3. **Test small first**:
   - Start with 10-20 subtitle entries
   - Once satisfied, run full batch

4. **Use secondary API key**:
   - If you have quota issues
   - Helps with large batch translations

## 📈 Expected Performance

- **Speed**: ~2-5 seconds per 100 entries (depends on model)
- **Quality**: Should match or exceed your Python translator
- **Reliability**: Structured JSON ensures consistency

## ✅ Success Checklist

- [ ] Code compiles without errors
- [ ] GUI loads and shows AI Translation tab
- [ ] Can select subtitle files
- [ ] Translation completes successfully
- [ ] Output file has correct format
- [ ] Translation quality matches or exceeds Python version
- [ ] No content gets blocked by safety filters
- [ ] Timing and structure perfectly preserved

---

**Ready to test?** Just run `./subtitle-forge` and try it out! 🚀
