import { defineStore } from "pinia";
import axios from "axios";

interface UserInfo {
  id: string;
  username: string;
  discriminator: string;
  avatar: string;
}

export const useUserInfoStore = defineStore("userinfo", {
  state: (): UserInfo => ({
    avatar: "",
    discriminator: "",
    id: "",
    username: "",
  }),
  actions: {
    async refreshUserInfo() {
      const { data } = await axios.get<UserInfo>("/oauth2/user");
      this.$patch(data);
    },

    async logout() {
      await axios.post("/oauth2/logout");
      this.$patch({
        avatar: "",
        discriminator: "",
        id: "",
        username: "",
      });
    },

    login() {
      window.location.href = "/oauth2/initiate";
    },
  },
  getters: {
    account(state) {
      return state.username + "#" + state.discriminator;
    },
  },
});
