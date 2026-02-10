from typing import Optional

from kfp import dsl, kubernetes


@dsl.component
def clone_github_repo(pvc_path: str, repo_url: str, target_dir: Optional[str] = None) -> str:
    '''
    Clone a GitHub repository using GitHub CLI
    :param pvc_path: PVC path where to clone the repository
    :param repo_url: GitHub repository URL
    :param target_dir: Target directory name (optional, defaults to repo name)
    :return: Path to cloned repository
    '''
    import subprocess
    import os

    # Change to PVC directory
    os.chdir(pvc_path)

    # Remove existing directory if it exists
    if os.path.exists(target_dir):
        subprocess.run(['rm', '-rf', target_dir], check=True)

    # Clone using GitHub CLI
    try:
        subprocess.run(['gh', 'repo', 'clone', repo_url, target_dir], check=True, env=os.environ.copy())
    except subprocess.CalledProcessError as e:
        raise RuntimeError(f"GitHub clone failed: {e}")

    # Return full path
    repo_path = os.path.join(pvc_path, target_dir)
    if not os.path.exists(repo_path):
        raise RuntimeError(f"Clone failed - directory not created: {repo_path}")

    return repo_path


@dsl.component(packages_to_install=["claude-agent-sdk>=0.1.0"])
def run_claude_session(pvc_path: str, user_prompt: str):
    '''
    Create a claude Client session and run a query
    :param pvc_path: Workspace PVC root path (use dsl.WORKSPACE_PATH_PLACEHOLDER).
    :param user_prompt: Prompt to query claude with
    :return:
    '''
    from claude_agent_sdk import ClaudeSDKClient, ClaudeAgentOptions, AssistantMessage, TextBlock, ThinkingBlock
    from typing import Any
    import uuid
    import asyncio

    # Generate Unique Session ID
    session_id = f"session-{uuid.uuid4().hex[:12]}"

    # Configure max claude session timeout
    timeout_seconds = "1800"
    print(f"Using AI timeout: {timeout_seconds}s for session {session_id}")

    def _init_sdk_client(self) -> Any:
        """Initialize and connect Claude SDK client using config.

        Returns:
            Connected Claude SDK client instance.
        """

        print(f"🚀 Initializing Claude SDK client...")

        # Load default system prompt from .claude/references/system_prompt.md
        system_prompt = None
        try:
            with open('/workspace/.claude/references/system_prompt.md', 'r') as f:
                system_prompt = f.read()
                print(f"📖 Loaded default system prompt from .claude/references/system_prompt.md")
        except FileNotFoundError:
            print(f"⚠️  Default system prompt file not found, using fallback")
            system_prompt = "You are a helpful assistant focused on software engineering and development."

        print(f"🎯 Using system prompt (first 100 chars): {system_prompt[:100]}...")


        # Configure options using config values
        print(f"⚙️  Configuring Claude SDK options...")
        options = ClaudeAgentOptions(
            system_prompt=system_prompt,
            permission_mode="bypassPermissions",  # type: ignore
            max_thinking_tokens=100_000,
            max_buffer_size=50_000_000
        )
        print(f"✅ Claude SDK options configured successfully")

        # Initialize SDK client
        print(f"🤖 Creating Claude SDK client instance...")
        sdk_client = ClaudeSDKClient(options=options)
        print(f"✅ Claude SDK client created successfully")

        return sdk_client

    async def make_call(
            self,
            sdk_client: ClaudeSDKClient,
            session_id: str,
            user_prompt: str,
    ) -> str:
        """Make a call to Claude with configurable timeout.

        Args:
            sdk_client: Claude SDK Client to use
            user_prompt: User's prompt/question.
            session_id: Existing session ID to continue. If None, uses current session.

        Returns:
            Claude's response as a string.

        Raises:
            asyncio.TimeoutError: If the request exceeds the configured timeout.
        """

        try:
            # Wrap the entire Claude interaction with timeout
            async with asyncio.timeout(timeout_seconds):
                # Ensure SDK client is connected before first use
                print(f"🔍 Starting Claude SDK query with session_id: {session_id}")
                messages: list = None

                try:
                    print(f"📡 Sending query to Claude SDK...")
                    await sdk_client.query(user_prompt, session_id=session_id)
                    print(f"✅ Query sent successfully to Claude SDK")

                except Exception as e:
                    try:
                        print(f"📡 Retrying query after initial error...")
                        await sdk_client.query(user_prompt, session_id=session_id)
                        print(f"✅ Query sent successfully after reconnection")
                    except Exception as e:
                        print(f"❌ CRITICAL: Even the retried query to Claude Agent failed")
                        raise RuntimeError("Failed to query", e)

                # Collect response with enhanced logging
                print(f"👂 Starting to receive response from Claude SDK...")
                response_parts = []
                message_count = 0

                try:
                    async for message in sdk_client.receive_response():
                        message_count += 1

                        if isinstance(message, AssistantMessage):
                            print(f"🤖 AssistantMessage with {len(message.content)} blocks")
                            for i, block in enumerate(message.content):
                                print(f"📦 Block {i}: {type(block).__name__}")
                                if isinstance(block, TextBlock):
                                    print(f"📝 TextBlock content length: {len(block.text)}")
                                    response_parts.append(block.text)
                                elif isinstance(block, ThinkingBlock):
                                    # Optionally include thinking blocks
                                    print(f"💭 ThinkingBlock content length: {len(block.thinking)}")
                        else:
                            print(f"⚠️  Non-AssistantMessage: {type(message)} - {message}")
                except Exception as response_error:
                    print(f"❌ Error during response reception: {response_error}")
                    print(f"📊 Messages received before error: {message_count}")
                    raise

                print(f"📊 Response collection complete - Total messages: {message_count}, response_parts: {len(response_parts)}")
                if message_count == 0:
                    print(f"❌ CRITICAL: No messages received from Claude SDK!")
                    return "No response received from Claude SDK. This may indicate MCP server connection issues or configuration problems."

        except asyncio.TimeoutError:
            print(f"Claude API call timed out after {timeout_seconds}s for session {session_id}")
            raise asyncio.TimeoutError(
                f"Claude API call timed out after {timeout_seconds}s. "
                f"You can increase the timeout with --timeout <seconds> or in your config."
            )

        response = "\n".join(response_parts)

        print(f"Response parts count: {len(response_parts)}, total length: {len(response)}")
        print(f"Response content preview: {repr(response[:200])}")

        return response

    sdk_client = _init_sdk_client()
    return make_call(sdk_client=sdk_client, user_prompt=user_prompt)

@dsl.pipeline(
    name="Claude Code Pipeline",
    description="Clone GitHub repo and run Claude analysis",
    pipeline_config=dsl.PipelineConfig(
        workspace=dsl.WorkspaceConfig(
            size='5Gi',
            kubernetes=dsl.KubernetesWorkspaceConfig(
                pvcSpecPatch={'storageClassName': 'standard', 'accessModes': ['ReadWriteMany']}
            ),
        ),
    ),
)
def claude_pipeline(
    user_prompt: str | list[str],
    repo_url: str | list[str],
    secret_name: str = "claude-git-secrets"
):
    """
    Pipeline that can clone a GitHub repository and run Claude analysis

    :param user_prompt: Prompt(s) for Claude queries
    :param repo_url: Optional GitHub repository URL(s) to clone
    """

    # Update user prompt to include repository context if repo_url provided
    if isinstance(user_prompt, list):
        assert isinstance(repo_url, list), "Please provide equal number of repositories to work with"
    if isinstance(repo_url, list) and isinstance(user_prompt, str):
        user_prompt = [user_prompt * len(repo_url)]
    if isinstance(repo_url, str) and isinstance(user_prompt, list):
        repo_url = [repo_url * len(user_prompt)]
    if isinstance(repo_url, str) and isinstance(user_prompt, str):
        repo_url = [ repo_url ]
        user_prompt = [ user_prompt ]


    with dsl.parallelFor(items=repo_url) as url:
        # Extract repository name if target_dir not provided
        target_dir: str = None
        repo_name = repo_url.split('/')[-1]
        if repo_name.endswith('.git'):
            repo_name = repo_name[:-4]
        target_dir = repo_name

        # Step 1: Clone repository
        clone_task = clone_github_repo(
            pvc_path=dsl.WORKSPACE_PATH_PLACEHOLDER,
            repo_url=repo_url,
            target_dir=target_dir
        )

        # Run Claude session
        claude_session = run_claude_session(
            pvc_path=dsl.WORKSPACE_PATH_PLACEHOLDER,
            user_prompt=clone_task.output
        )

        # Configure secrets for Claude session
        kubernetes.use_secret_as_env(
            claude_session,
            secret_name=secret_name
        )