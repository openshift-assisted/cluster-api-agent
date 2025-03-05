#!/usr/bin/env python3

import os
import logging
from github import Github, GithubIntegration

logger = logging.getLogger("github-auth")

def get_github_client():
    """
    Creates a PyGithub client using GitHub App authentication
    
    Environment variables:
        GITHUB_APP_ID: The ID of your GitHub App
        GITHUB_APP_INSTALLATION_ID: The installation ID for your GitHub App
        GITHUB_APP_PRIVATE_KEY: Private key .pem file
    """
    try:
        app_id = os.environ["GITHUB_APP_ID"]
        installation_id = os.environ["GITHUB_APP_INSTALLATION_ID"]
        private_key = os.environ["GITHUB_APP_PRIVATE_KEY"]
        
        if not all(bool(x) for x in [app_id, installation_id, private_key]):
            logger.error("Missing GitHub App credentials in environment")
            raise ValueError("""
                Missing GitHub App credentials.
                Ensure GITHUB_APP_ID, GITHUB_APP_INSTALLATION_ID, and GITHUB_APP_PRIVATE_KEY are set.
            """)
            
        integration = GithubIntegration(
            int(app_id),
            private_key
        )
        access_token = integration.get_access_token(int(installation_id))
        github_client = Github(access_token.token)
        logger.debug("Created GitHub client with new token")
        return github_client
    except Exception as e:
        logger.error(f"Error creating GitHub client: {e}")
        raise
