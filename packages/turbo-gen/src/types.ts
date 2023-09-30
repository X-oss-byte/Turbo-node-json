import type * as PlopTypes from "node-plop";

interface DependencyGroups {
  dependencies?: Record<string, string>;
  devDependencies?: Record<string, string>;
  peerDependencies?: Record<string, string>;
  optionalDependencies?: Record<string, string>;
}

interface PackageJson extends DependencyGroups {
  name: string;
  version: string;
  private?: boolean;
  description?: string;
  workspaces?: Array<string> | Record<string, Array<string>>;
  main?: string;
  module?: string;
  exports?: object;
  scripts?: Record<string, string>;
}

export type { PlopTypes, DependencyGroups, PackageJson };
