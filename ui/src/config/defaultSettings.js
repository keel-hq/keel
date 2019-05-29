// export default {
//   primaryColor: '#1890FF', // primary color of ant design
//   navTheme: 'light', // theme for nav menu
//   layout: 'topmenu', // nav menu position: sidemenu or topmenu
//   contentWidth: 'Fluid', // layout of content: Fluid or Fixed, only works when layout is topmenu
//   fixedHeader: false, // sticky header
//   fixSiderbar: false, // sticky siderbar
//   autoHideHeader: false, //  auto hide header
//   colorWeak: false,
//   multiTab: false,
//   production: process.env.NODE_ENV === 'production' && process.env.VUE_APP_PREVIEW !== 'true',
//   // vue-ls options
//   storageOptions: {
//     namespace: 'pro__', // key prefix
//     name: 'ls', // name variable Vue.[ls] or this.[$ls],
//     storage: 'local' // storage name session, local, memory
//   }
// }

export default {
  primaryColor: '#1890FF', // primary color of ant design
  navTheme: 'light', // theme for nav menu
  layout: 'topmenu', // nav menu position: sidemenu or topmenu
  contentWidth: 'Fluid', // layout of content: Fluid or Fixed, only works when layout is topmenu
  fixedHeader: false, // sticky header
  fixSiderbar: false, // sticky siderbar
  autoHideHeader: false, //  auto hide header
  colorWeak: false,
  multiTab: false,
  production: process.env.NODE_ENV === 'production' && process.env.VUE_APP_PREVIEW !== 'true',
  // vue-ls options
  storageOptions: {
    namespace: 'keel__',
    name: 'ls',
    storage: 'local'
  }
}
