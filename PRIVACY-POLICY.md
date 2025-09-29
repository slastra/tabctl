**Last Updated: September 28, 2024**

## Overview

TabCtl Browser Extension ("the Extension") is committed to protecting your privacy. This privacy policy explains how the Extension handles information when you use it to control browser tabs from the command line.

## Information Collection and Use

### What We Collect

The Extension **does not collect any personal information**.

The Extension only:
- Reads browser tab information (URLs, titles, tab IDs) when requested by the local TabCtl command-line tool
- Transmits this information locally to the TabCtl mediator process running on your computer
- Executes tab management commands (open, close, activate) as instructed by the local tool

### What We Don't Collect

The Extension does **not**:
- Collect browsing history
- Store any tab information
- Track user behavior
- Send data to external servers
- Use analytics or telemetry
- Create user profiles
- Share data with third parties
- Store cookies or local storage data

## Data Transmission

All communication occurs **locally on your computer** between:
1. The TabCtl command-line tool
2. The TabCtl mediator process
3. This browser extension

No data ever leaves your computer. The Extension uses Chrome's Native Messaging API to communicate exclusively with the locally installed TabCtl mediator process.

## Data Storage

The Extension does **not store any data**. Tab information is only accessed in real-time when you execute TabCtl commands and is immediately discarded after the command completes.

## Third-Party Services

The Extension does **not use any third-party services**, including:
- Analytics services
- Advertising networks
- Social media integrations
- Cloud storage services
- External APIs

## Permissions Used

The Extension requires certain permissions to function:

- **nativeMessaging**: To communicate with the local TabCtl command-line tool
- **tabs**: To read and manage browser tabs as commanded
- **activeTab**: To interact with the currently active tab
- **<all_urls>**: To display complete tab information for all websites

These permissions are used solely for the stated functionality and not for data collection.

## Open Source

TabCtl is open source software. You can review the complete source code at:
https://github.com/slastra/tabctl

## Children's Privacy

The Extension does not collect information from anyone, including children under 13 years of age.

## Updates to This Policy

Any updates to this privacy policy will be posted in the project repository and updated in the Chrome Web Store listing. The "Last Updated" date at the top will be revised accordingly.

## Security

Since the Extension doesn't collect or transmit data externally, there are no servers or databases that could be compromised. All operations are performed locally on your machine.

## Your Rights

Since we don't collect any data, there is no personal information to access, correct, or delete. You can uninstall the Extension at any time through your browser's extension management page.

## Contact Information

For questions or concerns about this privacy policy or the Extension, please:
- Open an issue on GitHub: https://github.com/slastra/tabctl/issues
- Review the documentation: https://github.com/slastra/tabctl

## Compliance

This Extension complies with:
- Chrome Web Store Developer Program Policies
- General Data Protection Regulation (GDPR) - by not collecting any personal data
- California Consumer Privacy Act (CCPA) - by not collecting any personal information

## Consent

By installing and using the TabCtl Browser Extension, you acknowledge that you have read and understood this privacy policy.

---

**Remember**: TabCtl is a local tool. Your browsing data never leaves your computer.
