export function isUrlCanPreview(url) {
    if (url.includes('github.com/')) {
        const [owner, repo, type, number] = url.split('github.com/')[1].split('/');
        return !(!owner | !repo | !type | !number);
    }
    return false;
}
