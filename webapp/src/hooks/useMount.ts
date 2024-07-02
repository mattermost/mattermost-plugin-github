import {useEffect} from 'react';

export const useMount = (callback: () => void) => {
    useEffect(() => {
        callback();
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);
};
