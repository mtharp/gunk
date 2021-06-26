import { Module, VuexModule, MutationAction, getModule, Action } from 'vuex-module-decorators';
import axios from "axios";
import store from '.';

interface UserInfo {
    id: string;
    username: string;
    discriminator: string;
    avatar: string;
}

@Module({ name: "userinfo", store: store, dynamic: true })
export class UserInfoModule extends VuexModule {
    user: UserInfo = {id: "", username: "", discriminator: "", avatar: ""};

    @MutationAction
    async refreshUserInfo() {
        const { data } = await axios.get<UserInfo>('/oauth2/user');
        return {user: data};
    }

    @MutationAction
    async logout() {
        await axios.post('/oauth2/logout');
        return { user: { id: "", username: "", discriminator: "", avatar: "" }};
    }

    @Action
    login() {
        window.location.href = '/oauth2/initiate';
    }

    get account() {
        return this.user.username + "#" + this.user.discriminator;
    }
    get avatar() { return this.user.avatar; }
}

export default getModule(UserInfoModule);
