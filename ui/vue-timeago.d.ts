import Vue from "vue";


export interface Options {
    locale?: string;
}

export default function install(v: typeof Vue, opts: Options): void;
