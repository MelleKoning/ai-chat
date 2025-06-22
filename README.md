# ai-chat

The code repository ai-chat is to setup a console chat application that enables simple text interaction with the GEMINI API.
The `/cmd/tviewchat` folder contains the main entry point to run against a cloud model.

## Pre conditions

To use this program, you need a valid API key to access Googles Gemini API endpoint. See links below for references

## Concept of accessing a model via an API

The code folders in `/cmd` are aimed for a quick trial on getting access to a google Gemini model in the cloud. The code should be self explanatory.

References:

- <https://ai.google.dev/gemini-api/docs/quickstart?lang=go>
- <https://ai.google.dev/gemini-api/docs/text-generation>
- <https://www.mellekoning.nl/king-julian-can-code/>

## TviewChat application

To have a good chat rendered in the console the code is now using "tview" as a library. The chat can be controlled by typing a command in the bottom part of the screen and using TAB to go to the SUBMIT button. When submitting the command, the command will be send to the backend gemini API, and the response is being rendered in the outputView at the top.

![Tview chat in console](/docs/demo.gif)

The chat window will have the full history of the chat and when selected (has the focus) you can simply scroll up/down through the chat. Your own commands are shown in green at the moment.

The returned responses from the API are rendered via "glamour", which enables nice colourization of example code and markdown.

### Analyzing git diff with a prompt

The code can now a "git diff" that you can generate from a git repository.

First, generate a file `gitdiff.txt` with a command like this

```bash
git diff -U10 cd71..HEAD ':!vendor' > gitdiff.txt
```

or to get changes of a branch against master:

```bash
git diff -U10 master..branchname ':!vendor' > gitdiff.txt
```

Explanation: the hashes are examples from two consecutive git hashes found when
simply doing a "git log" statement. Put the oldest hash first so that added lines get a + and removed lines get a -, or you get it backwards. note that the `-- . ':! vendor'` part is to ignore the vendor folder, as we are only interested in actual updates of changes from the authors of the repository.

You could also inspect changes that are not even committed yet by looking at git staged changes:

```bash
 git diff --staged -U10 ':!vendor' > gitdiff.txt
 ```

This way, you can review your code, update and enhance it before committing!

### Run the chat tool

```bash
> go run ./cmd/tviewchat/main.go
```

You can TAB to choose a systemPrompt. You can start a chat, but the goal is to choose "Reviewfile" in the dropdown.
When you select that, the file-contents "gitdiff.txt" will be send to the gemini API for analyses, call the cloud API and show suggestions for the diff.

#### Storing chats

Added is the ability to store chats as history files. This is because the Gemini API is capable of a huge context window, so that you can later load the chat-history back and continue the conversation.


Chats are stored in your home config folder, usually `~/.config/ai-chat/history`

## Navigating the Console User Interface (CUI)

Navigation in the UI goes via a few default keys:

- TAB should switch focus to another GUI item
- In the outputview, where model responses are shown, you can press ENTER to get and use the responses. That is When the AI model generated example code that you might want to try out, you can press ENTER to change the focus of the view and select any of the examples for copying. Pressing ESC returns to the default view to continue the chat
- The dropdown is currently used for additional features like storing and loading chats and exiting the program
- The inputbox at the bottom of the page is to type in your prompts for the modal. Type TAB and click ENTER when the SUBMIT button has the focus to send your prompt to the model in the cloud - then pleae be a bit patience awaiting the
   response which will be generated in the outputView at the top of the screen


## Features not yet implemented

There are several ideas to extend the code with some new features

- Change the glamour model dynamically for other default colours
- Cut down the history items as it seems there is a limit when sending history items
- Dynamically choosing other Gemini models instead of hardcoded modelstring
- More unit testing (oops) to assert the interaction of the model implementation and tview console app

## References

[www.mellekoning.nl](http://www.mellekoning.nl/)
