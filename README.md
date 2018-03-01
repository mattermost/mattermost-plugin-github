# mattermost-github
Mattermost plugin for Github

# Plan
- Features
    - Slash command to subscribe a channel to a repository
        - Filters as options on slash command
    - Link previews for:
        - Code snipits
        - Issues
        - PRs
    - Interactive dynamic posts for PRs/issues
        - Everything you can do in Github you can do from MM
        - See lables/assignee/fix-version/etc.
        - Inline code for small PRs?
        - Actions: Merge/close/change assignee/remove lables/Approve/requetchanges
        - Links: PR main page, direct to code
    - Comment on github is comment in MM and vice versa
    - Daily reminders for PR reviews.
    - Auto ticket linking. Replace pasted links with full iteractive post.

- Register user token
    - Slash command? 
        - Problem is that slash commands have history
        - Add ability to not record a slash command in history?
    - Oauth
    - Replying when we don't have token
        - Block reply?

- Two modes of operation, authenticated and inauthenticated.
    - Inauthenticated
        - No actions are available
        - No code previews
        - No private repsitories
        - All posts that have disabled content have a link to how to setup a github token to work with the plugin
    - Authenticated
        - User can perform ations and see code from private repsitories that they have access to. (server makes requests to github as them?)
