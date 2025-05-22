package prompts

type Prompt struct {
	Name   string
	Prompt string
}

var PromptList = []Prompt{
	{Name: "gitreview prompt",
		Prompt: `You are an expert developer and git super user. You do code reviews based on the git diff output between two commits.

	* The diff contains a few unchanged lines of code. Focus on the code that changed. Changed are added and removed lines.

	* The added lines start with a "+" and the removed lines that start with a "-"
	Complete the following tasks, and be extremely critical and precise in your review:

	* [Description] Describe the code change.

	* [Obvious errors] Look for obvious errors in the code and suggest how to fix.

	* [Improvements] Suggest improvements where relevant. Suggestions must be rendered as code, not as diff.

	* [Friendly advice] Give some friendly advice or heads up where relevant.

	* [Stop when done] Stop when you are done with the review.
`},
	{

		Name: "gitreview prompt - only top 2",
		Prompt: `Please perform a thorough code review of the following git diff. Your review should address the following top 6 tasks:
**Task 1:  Correctness and Error Handling**

* Analyze for logical errors, bugs, and regressions.
* Evaluate handling of edge cases and error conditions.
* Confirm alignment with described purpose/context.

**Task 2:  Code Quality and Readability**

* Assess clarity, simplicity, and understandability.
* Evaluate naming (variables, functions, classes).
* Check for necessary and clear comments.
* Identify redundant/unnecessary code.
* Note any stylistic inconsistencies *within the diff*.

**Task 3:  Object-Oriented Principles (where applicable)**

* Evaluate appropriate use of classes, objects, and methods.
* Identify any violations of OO principles (e.g., single responsibility, open/closed principle, etc.).
* Suggest refactoring towards a more OO design if procedural code is present where OO is more suitable.

**Task 4:  Clean Code Practices**

* Apply clean code principles (e.g., keep functions small, do one thing, use descriptive names).
* Assess for code duplication and suggest DRY (Don't Repeat Yourself) principle.
* Evaluate for KISS (Keep It Simple, Stupid) principle.

**Task 5:  Performance and Security**

* Analyze potential performance bottlenecks.
* Identify potential security vulnerabilities *within the diff*.

**Task 6:  Testability and Test Implications**

* Assess the impact of changes on testability.
* Determine the need for new/modified tests.

Provide feedback organized by task, referencing specific lines. Explain your reasoning for each issue and suggestion.

Before answering take the top 2 suggestions as response. Do not respond more than 2 suggestions.
`,
	},
	{
		Name: "gitreview actionable prompt - Address top 2 tasks in each category",
		Prompt: `Please perform a focused code review of the following git diff, providing specific examples  Address the top 2 tasks in each category:

**Context:**

* Brief description of the purpose and context of these changes:
* Relevant background information or related issues:
* Any specific areas you would like the reviewer to pay particular attention to:

**Review Tasks:**

**1. Correctness & Logic:**

* 1.  Identify potential logical errors or bugs introduced in the diff.
* 2.  Analyze the handling of specific edge cases or error conditions modified by the diff.

**2. Readability & Style:**

* 1.  Assess if the diff makes the code clearer or more confusing (provide specific examples of improved or worsened clarity).
* 2.  Evaluate the naming of new variables/functions in the diff for descriptiveness and consistency.

**3. OO Principles & Design:**

* 1.  Identify any changes in the diff that violate basic OO principles (e.g., a method doing too much, tight coupling).
* 2.  If the diff introduces procedural code, suggest *specific* refactoring steps within the scope of the diff to improve OO design.

**4. Clean Code:**

* 1.  Point out any code duplication introduced or not addressed by the diff.
* 2.  Assess if new functions/methods in the diff adhere to the "single responsibility principle".

**5. Performance & Security:**

* 1.  Identify any *obvious* performance regressions introduced by the diff (e.g., inefficient loops, excessive object creation).
* 2.  Flag any *clear* security vulnerabilities added in the diff (e.g., lack of input validation).

**6. Testing:**

* 1.  Determine if the changes in the diff clearly require new or modified unit tests.
* 2.  Note any existing tests modified or removed by the diff and assess their relevance.

Provide your review organized by category, with detailed code examples, to illustrate issues and suggestions.
			`,
	},
	{
		Name: "concise prompt - code optimization focused - before and after changes",
		Prompt: `Please provide a code optimization-focused review of the following git diff. Provide "before" and "after" code snippets to illustrate each suggestion.

**Context:**

* Brief description of the purpose and context of these changes:
* Relevant background information:

**Optimization Targets (Focus your review on these):**

* Performance
* Code Duplication
* Maintainability

**Review Tasks:**

1.  **Performance Optimization:**
    * Identify any changes that introduce performance regressions or limit potential optimizations.
    * Suggest code-level optimizations to improve performance (provide "before" and "after" code).

2.  **Code Duplication & Maintainability:**
    * Find any code duplication introduced or opportunities to reduce existing duplication for better maintainability.
    * Suggest refactoring steps (with code examples) to apply the DRY principle.

3.  **Optimization-Enabling Refactoring:**
    * Identify sections of code that, if refactored, would open up further optimization possibilities.
    * Provide refactoring suggestions (with code examples) that set the stage for future optimizations.

4.  **Testability Impact:**
    * Assess if the changes make the code harder or easier to test.
    * Suggest optimizations that also improve testability.

Provide detailed explanations for each optimization suggestion, with "before" and "after" code snippets.
`,
	},
	{
		Name: "diff refactoring focus - DRY, SOLID",
		Prompt: `Please provide a refactoring-focused review of the following git diff, with detailed "before" and "after" code examples *within the scope of the diff*.

**Context:**

* Brief description of the purpose and context of these changes:
* Relevant background information:

**Important:** Remember that you are reviewing a *diff*. "Before" code should represent the original code *as shown in the diff* (the "-" lines), and "after" code should represent the changed code *as shown in the diff* (the "+" lines), incorporating refactoring suggestions.

**Refactoring Goals:**

 * DRY: Don't repeat yourself principle
 * Smaller, Single-Responsibility Functions
 * Open closed principle
 * Liskov Substitution Principle
 * Interface segregation
 * Dependency Inversion principle
 * Enhanced Object-Oriented Design

**Review Tasks:**

1.  **Function Size within the Diff:**

    * Identify functions *modified or introduced in the diff* that become too large or complex *after the changes*.
    * Provide refactoring suggestions with "before" and "after" code examples (from the diff) to break down these functions.

2.  **OO Opportunities in the Changed Code:**

    * Analyze the *changes in the diff* for opportunities to introduce new classes or objects to better encapsulate data and behavior *within the scope of the diff*.
    * If the *diff introduces* procedural code patterns, suggest refactoring steps (with code examples from the diff) to shift towards an object-oriented approach.

3.  **Function Naming in the Diff:**

    * Evaluate the naming of functions *modified or added in the diff*.
    * Suggest refactoring examples *within the diff* to improve function names for brevity and clarity, especially if made possible by Task 1.

4.  **Code Organization Changes for OO:**

    * Assess if the *diff* introduces code that could be better organized within existing or new classes *within the scope of the diff*.
    * Provide refactoring suggestions with code examples (from the diff) to achieve better code organization and encapsulation.

Provide detailed explanations for each refactoring suggestion, with clear "before" and "after" code snippets *from the diff*.
Show suggested code as code, not as diff.
`,
	},
}
