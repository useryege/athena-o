import {AccountsService} from './accounts-service';
import {ApplicationsService} from './applications-service';
import {AthenaApplicationService} from './athena-application-service';
import {AuthService} from './auth-service';
import {CertificatesService} from './cert-service';
import {ClustersService} from './clusters-service';
import {ExtensionsService} from './extensions-service';
import {GnuPGPublicKeyService} from './gpgkey-service';
import {NotificationService} from './notification-service';
import {ProjectsService} from './projects-service';
import {RepositoriesService} from './repo-service';
import {RepoCredsService} from './repocreds-service';
import {UserService} from './user-service';
import {VersionService} from './version-service';
import {ViewPreferencesService} from './view-preferences-service';
import {WormService} from './worm-service';
export interface Services {
    athenaApplication: AthenaApplicationService;
    applications: ApplicationsService;
    users: UserService;
    authService: AuthService;
    certs: CertificatesService;
    repocreds: RepoCredsService;
    repos: RepositoriesService;
    clusters: ClustersService;
    projects: ProjectsService;
    viewPreferences: ViewPreferencesService;
    version: VersionService;
    accounts: AccountsService;
    gpgkeys: GnuPGPublicKeyService;
    extensions: ExtensionsService;
    worm: WormService;
    notification: NotificationService;
}

export const services: Services = {
    athenaApplication: new AthenaApplicationService(),
    applications: new ApplicationsService(),
    authService: new AuthService(),
    clusters: new ClustersService(),
    users: new UserService(),
    certs: new CertificatesService(),
    repos: new RepositoriesService(),
    repocreds: new RepoCredsService(),
    projects: new ProjectsService(),
    viewPreferences: new ViewPreferencesService(),
    version: new VersionService(),
    accounts: new AccountsService(),
    gpgkeys: new GnuPGPublicKeyService(),
    extensions: new ExtensionsService(),
    worm: new WormService(),
    notification: new NotificationService()
};

export * from './projects-service';
export * from './view-preferences-service';
