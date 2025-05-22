# aifun

The code repository aifun is to play around with several AI thingies. The `/cmd` contains some GoLang code to run against a cloud model, and the `docker-compose.yml` is to run certain models locally using Meta's Ollama.

## Concept of accessing a model via an API

The code folders in `/cmd` are aimed for a quick trial on getting access to a google Gemini model in the cloud. The code should be self explanatory.

References:

- <https://ai.google.dev/gemini-api/docs/quickstart?lang=go>
- <https://ai.google.dev/gemini-api/docs/text-generation>
- <https://www.mellekoning.nl/king-julian-can-code/>

## TviewChat application

To have a good chat rendered in the console the code is now using "tview" as a library. The chat can be controlled by typing a command in the bottom part of the screen and using TAB to go to the SUBMIT button. When submitting the command, the command will be send to the backend gemini, and the response is being rendered in the outputView at the top.

![Tview chat in console](/docs/demochat.png)

The chat window will have the full history of the chat and when selected (has the focus) you can simply scroll up/down through the chat. Your own commands are shown in green at the moment.

### Analyzing git diff with a prompt

The code can now analyze a "git diff" that you can generate from a git repository.

First, generate a file `gitdiff.txt` with a command like this

```bash
git diff -U10 88217..2042eb ':!vendor' > gitdiff.txt
```

(the above command should work on this repository)

Explanation: the hashes are examples from two consecutive git hashes found when
simply doing a "git log" statement. Put the oldest hash first so that added lines get a + and removed lines get a -, or you get it backwards. note that the `-- . ':! vendor'` part is to ignore the vendor folder, as we are only interested in actual updates of changes from the authors of the repository.

Note that if you are on a branch and want to get changes from your latest commit to the branch with master, use:

```bash
git diff -U10 master..HEAD -- . ':!vendor' > gitdiff.txt
```

Run the gitdiff tool

> go run ./cmd/diffreviewer/main.go

You will be presented with a choice for a systemPrompt. You can start a chat, but the goal is to type "file".
When you type "file" the code will read the "gitdiff.txt" for analyses, call the cloud API and show suggestions for the diff.

## Docker-compose ollama and web UI

The idea of the `docker-compose.yaml` file is to have a singular way of starting ollama and openwebui.

The volume mapping is to ensure the pulled models are shared whenever we would start ollama from the commandline in linux.

## startup command

```bash
docker-compose up
```

Then open a browser and go to the url:

```bash
http://localhost:20080
```

The above commands will startup ollama and openwebui and keep the console open. This way, you can see the incoming HTTP requests that openwebui is executing against ollama.

Especially when using certain tools, like websearch, within openwebui, it is nice to see what requests ollama and openwebui are executing "under the hood". Sometimes requests fail and all of the logging that the docker containers are doing makes it a bit more clear what is happening when `openwebui` is exercising requests with `ollama`, and how the diferent LLM models are being exercised by `ollama`. For example, at startup, you can see if `ollama` was able to find NVIDIA hardware, as if not, it will fall back to CPU which makes performance far worse.

## Trouble shooting

Sometimes starting up the docker-compose does not work because another ollama instance is already running on your system. This is not surprising as ollama does have start commands, but does not have stop commands. Essentially, the ollama tool is a client tool that ensures that "ollama serve" is running, and when it does, it does not stop (ref: version that was used so far does not have a stop command for the server)

Investigate what other instance of the ollama server is running on port 11434 with

`sudo netstat -peanut | grep ollama`

you can kill other ollama processes or try to stop those via several commands

To stop a server that runs with ubuntu or linux mint service:

```bash
systemctl stop ollama
```

to stop an already running LLM:

```bash
ollama stop
```

to stop an ollama docker instance, you have to lookup the docker-id of the ollama instance and stop and or remove it:

```bash
docker ps -a
docker stop <ollama docker instace>
docker rm <ollama docker instance>
```

## References

[www.mellekoning.nl](http://www.mellekoning.nl/)
