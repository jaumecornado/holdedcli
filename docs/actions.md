# Holded Actions Catalog (Snapshot)

Generated from `holded actions list --json`.

- Source: https://developers.holded.com/reference/api-key
- Generated at (UTC): 2026-02-14T17:33:01Z
- Total actions: 136

This file is a versioned snapshot for skills and offline reference.
Runtime discovery is still available via `holded actions list`.

## Accounting API

- `accounting.createaccount` | `POST /api/accounting/v1/account` | operation: `createAccount`
- `accounting.createentry` | `POST /api/accounting/v1/entry` | operation: `createEntry`
- `accounting.listaccounts` | `GET /api/accounting/v1/chartofaccounts` | operation: `listaccounts`
- `accounting.listdailyledger` | `GET /api/accounting/v1/dailyledger` | operation: `listDailyLedger`

## CRM API

- `crm.cancel-booking` | `DELETE /api/crm/v1/bookings/{bookingId}` | operation: `Cancel Booking`
- `crm.create-booking` | `POST /api/crm/v1/bookings` | operation: `Create Booking`
- `crm.create-event` | `POST /api/crm/v1/events` | operation: `Create Event`
- `crm.create-funnel` | `POST /api/crm/v1/funnels` | operation: `Create Funnel`
- `crm.create-lead` | `POST /api/crm/v1/leads` | operation: `Create Lead`
- `crm.create-lead-note` | `POST /api/crm/v1/leads/{leadId}/notes` | operation: `Create Lead Note`
- `crm.create-lead-task` | `POST /api/crm/v1/leads/{leadId}/tasks` | operation: `Create Lead Task`
- `crm.delete-event` | `DELETE /api/crm/v1/events/{eventId}` | operation: `Delete Event`
- `crm.delete-funnel` | `DELETE /api/crm/v1/funnels/{funnelId}` | operation: `Delete Funnel`
- `crm.delete-lead` | `DELETE /api/crm/v1/leads/{leadId}` | operation: `Delete Lead`
- `crm.delete-lead-task` | `DELETE /api/crm/v1/leads/{leadId}/tasks` | operation: `Delete Lead Task`
- `crm.get-available-slots-for-location` | `GET /api/crm/v1/bookings/locations/{locationId}/slots` | operation: `Get available slots for location`
- `crm.get-booking` | `GET /api/crm/v1/bookings/{bookingId}` | operation: `Get Booking`
- `crm.get-event` | `GET /api/crm/v1/events/{eventId}` | operation: `Get Event`
- `crm.get-funnel` | `GET /api/crm/v1/funnels/{funnelId}` | operation: `Get Funnel`
- `crm.get-lead` | `GET /api/crm/v1/leads/{leadId}` | operation: `Get Lead`
- `crm.list-bookings` | `GET /api/crm/v1/bookings` | operation: `List Bookings`
- `crm.list-events` | `GET /api/crm/v1/events` | operation: `List Events`
- `crm.list-funnels` | `GET /api/crm/v1/funnels` | operation: `List Funnels`
- `crm.list-leads` | `GET /api/crm/v1/leads` | operation: `List Leads`
- `crm.list-locations` | `GET /api/crm/v1/bookings/locations` | operation: `List Locations`
- `crm.update-booking` | `PUT /api/crm/v1/bookings/{bookingId}` | operation: `Update Booking`
- `crm.update-event` | `PUT /api/crm/v1/events/{eventId}` | operation: `Update Event`
- `crm.update-funnel` | `PUT /api/crm/v1/funnels/{funnelId}` | operation: `Update Funnel`
- `crm.update-lead` | `PUT /api/crm/v1/leads/{leadId}` | operation: `Update Lead`
- `crm.update-lead-creation-date` | `PUT /api/crm/v1/leads/{leadId}/dates` | operation: `Update Lead Creation Date`
- `crm.update-lead-note` | `PUT /api/crm/v1/leads/{leadId}/notes` | operation: `Update Lead Note`
- `crm.update-lead-stage` | `PUT /api/crm/v1/leads/{leadId}/stages` | operation: `Update Lead Stage`
- `crm.update-lead-task` | `PUT /api/crm/v1/leads/{leadId}/tasks` | operation: `Update Lead Task`

## Invoice API

- `invoice.attach-file` | `POST /api/invoicing/v1/documents/{docType}/{documentId}/attach` | operation: `Attach File`
- `invoice.create-contact` | `POST /api/invoicing/v1/contacts` | operation: `Create Contact`
- `invoice.create-contact-group` | `POST /api/invoicing/v1/contacts/groups` | operation: `Create Contact Group`
- `invoice.create-document` | `POST /api/invoicing/v1/documents/{docType}` | operation: `Create Document`
- `invoice.create-expenses-account` | `POST /api/invoicing/v1/expensesaccounts` | operation: `Create Expenses Account`
- `invoice.create-numbering-serie` | `POST /api/invoicing/v1/numberingseries/{type}` | operation: `Create Numbering Serie`
- `invoice.create-payment` | `POST /api/invoicing/v1/payments` | operation: `Create Payment`
- `invoice.create-product` | `POST /api/invoicing/v1/products` | operation: `Create Product`
- `invoice.create-sales-channel` | `POST /api/invoicing/v1/saleschannels` | operation: `Create Sales Channel`
- `invoice.create-service` | `POST /api/invoicing/v1/services` | operation: `Create Service`
- `invoice.create-treasury` | `POST /api/invoicing/v1/treasury` | operation: `Create Treasury`
- `invoice.create-warehouse` | `POST /api/invoicing/v1/warehouses` | operation: `Create Warehouse`
- `invoice.delete-contact` | `DELETE /api/invoicing/v1/contacts/{contactId}` | operation: `Delete Contact`
- `invoice.delete-contact-group` | `DELETE /api/invoicing/v1/contacts/groups/{groupId}` | operation: `Delete Contact Group`
- `invoice.delete-document` | `DELETE /api/invoicing/v1/documents/{docType}/{documentId}` | operation: `Delete Document`
- `invoice.delete-expenses-account` | `DELETE /api/invoicing/v1/expensesaccounts/{expensesAccountId}` | operation: `Delete Expenses account`
- `invoice.delete-numbering-serie` | `DELETE /api/invoicing/v1/numberingseries/{type}/{numberingSeriesId}` | operation: `Delete Numbering Serie`
- `invoice.delete-payment` | `DELETE /api/invoicing/v1/payments/{paymentId}` | operation: `Delete Payment`
- `invoice.delete-product` | `DELETE /api/invoicing/v1/products/{productId}` | operation: `Delete Product`
- `invoice.delete-sales-channel` | `DELETE /api/invoicing/v1/saleschannels/{salesChannelId}` | operation: `Delete Sales Channel`
- `invoice.delete-service` | `DELETE /api/invoicing/v1/services/{serviceId}` | operation: `Delete Service`
- `invoice.delete-warehouse` | `DELETE /api/invoicing/v1/warehouses/{warehouseId}` | operation: `Delete Warehouse`
- `invoice.get-api-invoicing-v1-products-productid-image-imagefilename` | `GET /api/invoicing/v1/products/{productId}/image/{imageFileName}`
- `invoice.get-attachment` | `GET /api/invoicing/v1/contacts/{contactId}/attachments/get` | operation: `Get attachment`
- `invoice.get-attachments-list` | `GET /api/invoicing/v1/contacts/{contactId}/attachments/list` | operation: `Get attachments list`
- `invoice.get-contact` | `GET /api/invoicing/v1/contacts/{contactId}` | operation: `Get Contact`
- `invoice.get-contact-group` | `GET /api/invoicing/v1/contacts/groups/{groupId}` | operation: `Get Contact Group`
- `invoice.get-expenses-account` | `GET /api/invoicing/v1/expensesaccounts/{expensesAccountId}` | operation: `Get Expenses Account`
- `invoice.get-numbering-series` | `GET /api/invoicing/v1/numberingseries/{type}` | operation: `Get Numbering Series`
- `invoice.get-payment` | `GET /api/invoicing/v1/payments/{paymentId}` | operation: `Get Payment`
- `invoice.get-product` | `GET /api/invoicing/v1/products/{productId}` | operation: `Get product`
- `invoice.get-product-image` | `GET /api/invoicing/v1/products/{productId}/image` | operation: `Get Product Image`
- `invoice.get-remittance` | `GET /api/invoicing/v1/remittances/{remittanceId}` | operation: `Get Remittance`
- `invoice.get-sales-channel` | `GET /api/invoicing/v1/saleschannels/{salesChannelId}` | operation: `Get Sales Channel`
- `invoice.get-service` | `GET /api/invoicing/v1/services/{serviceId}` | operation: `Get Service`
- `invoice.get-treasury` | `GET /api/invoicing/v1/treasury/{treasuryId}` | operation: `Get Treasury`
- `invoice.get-warehouse` | `GET /api/invoicing/v1/warehouses/{warehouseId}` | operation: `Get Warehouse`
- `invoice.getdocument` | `GET /api/invoicing/v1/documents/{docType}/{documentId}` | operation: `getDocument`
- `invoice.getdocumentpdf` | `GET /api/invoicing/v1/documents/{docType}/{documentId}/pdf` | operation: `GetDocumentPDF`
- `invoice.gettaxes` | `GET /api/invoicing/v1/taxes` | operation: `getTaxes`
- `invoice.list-contact-groups` | `GET /api/invoicing/v1/contacts/groups` | operation: `List Contact Groups`
- `invoice.list-contacts` | `GET /api/invoicing/v1/contacts` | operation: `List Contacts`
- `invoice.list-documents` | `GET /api/invoicing/v1/documents/{docType}` | operation: `List Documents`
- `invoice.list-expenses-accounts` | `GET /api/invoicing/v1/expensesaccounts` | operation: `List Expenses Accounts`
- `invoice.list-payment-methods` | `GET /api/invoicing/v1/paymentmethods` | operation: `List Payment methods`
- `invoice.list-payments` | `GET /api/invoicing/v1/payments` | operation: `List Payments`
- `invoice.list-product-images` | `GET /api/invoicing/v1/products/{productId}/imagesList` | operation: `List Product Images`
- `invoice.list-products` | `GET /api/invoicing/v1/products` | operation: `List Products`
- `invoice.list-products-stock` | `GET /api/invoicing/v1/warehouses/{warehouseId}/stock` | operation: `List products stock`
- `invoice.list-remittances` | `GET /api/invoicing/v1/remittances` | operation: `List Remittances`
- `invoice.list-sales-channels` | `GET /api/invoicing/v1/saleschannels` | operation: `List Sales Channels`
- `invoice.list-services` | `GET /api/invoicing/v1/services` | operation: `List Services`
- `invoice.list-treasuries` | `GET /api/invoicing/v1/treasury` | operation: `List Treasuries`
- `invoice.list-warehouses` | `GET /api/invoicing/v1/warehouses` | operation: `List Warehouses`
- `invoice.pay-document` | `POST /api/invoicing/v1/documents/{docType}/{documentId}/pay` | operation: `Pay Document`
- `invoice.send-document` | `POST /api/invoicing/v1/documents/{docType}/{documentId}/send` | operation: `Send Document`
- `invoice.ship-all-items` | `POST /api/invoicing/v1/documents/salesorder/{documentId}/shipall` | operation: `Ship all items`
- `invoice.ship-items-by-line` | `POST /api/invoicing/v1/documents/salesorder/{documentId}/shipbylines` | operation: `Ship items by line`
- `invoice.shipped-units-by-item` | `GET /api/invoicing/v1/documents/{docType}/{documentId}/shippeditems` | operation: `Shipped units by item`
- `invoice.update-contact` | `PUT /api/invoicing/v1/contacts/{contactId}` | operation: `Update Contact`
- `invoice.update-contact-group` | `PUT /api/invoicing/v1/contacts/groups/{groupId}` | operation: `Update Contact Group`
- `invoice.update-document` | `PUT /api/invoicing/v1/documents/{docType}/{documentId}` | operation: `Update Document`
- `invoice.update-document-pipeline` | `POST /api/invoicing/v1/documents/{docType}/{documentId}/pipeline/set` | operation: `Update document pipeline`
- `invoice.update-expenses-account` | `PUT /api/invoicing/v1/expensesaccounts/{expensesAccountId}` | operation: `Update Expenses Account`
- `invoice.update-numbering-serie` | `PUT /api/invoicing/v1/numberingseries/{type}/{numberingSeriesId}` | operation: `Update Numbering Serie`
- `invoice.update-payment` | `PUT /api/invoicing/v1/payments/{paymentId}` | operation: `Update Payment`
- `invoice.update-product` | `PUT /api/invoicing/v1/products/{productId}` | operation: `Update Product`
- `invoice.update-product-stock` | `PUT /api/invoicing/v1/products/{productId}/stock` | operation: `Update Product stock`
- `invoice.update-sales-channel` | `PUT /api/invoicing/v1/saleschannels/{salesChannelId}` | operation: `Update Sales Channel`
- `invoice.update-service` | `PUT /api/invoicing/v1/services/{serviceId}` | operation: `Update Service`
- `invoice.update-tracking-info` | `POST /api/invoicing/v1/documents/{docType}/{documentId}/updatetracking` | operation: `Update tracking info`
- `invoice.update-warehouse` | `PUT /api/invoicing/v1/warehouses/{warehouseId}` | operation: `Update Warehouse`

## Projects API

- `projects.create-project` | `POST /api/projects/v1/projects` | operation: `Create Project`
- `projects.create-project-time` | `POST /api/projects/v1/projects/{projectId}/times` | operation: `Create Project Time`
- `projects.create-task` | `POST /api/projects/v1/tasks` | operation: `Create Task`
- `projects.delete-project` | `DELETE /api/projects/v1/projects/{projectId}` | operation: `Delete Project`
- `projects.delete-project-time` | `DELETE /api/projects/v1/projects/{projectId}/times/{timeTrackingId}` | operation: `Delete Project Time`
- `projects.delete-task` | `DELETE /api/projects/v1/tasks/{taskId}` | operation: `Delete Task`
- `projects.get-api-projects-v1-projects-projectid-summary` | `GET /api/projects/v1/projects/{projectId}/summary`
- `projects.get-project` | `GET /api/projects/v1/projects/{projectId}` | operation: `Get Project`
- `projects.get-project-times` | `GET /api/projects/v1/projects/{projectId}/times` | operation: `Get Project Times`
- `projects.get-task` | `GET /api/projects/v1/tasks/{taskId}` | operation: `Get Task`
- `projects.getprojecttimes` | `GET /api/projects/v1/projects/{projectId}/times/{timeTrackingId}` | operation: `GetProjectTimes`
- `projects.list-projects` | `GET /api/projects/v1/projects` | operation: `List Projects`
- `projects.list-tasks` | `GET /api/projects/v1/tasks` | operation: `List Tasks`
- `projects.list-times` | `GET /api/projects/v1/projects/times` | operation: `List Times`
- `projects.update-project` | `PUT /api/projects/v1/projects/{projectId}` | operation: `Update Project`
- `projects.update-project-time` | `PUT /api/projects/v1/projects/{projectId}/times/{timeTrackingId}` | operation: `Update Project Time`

## Team API

- `team.createemployee` | `POST /api/team/v1/employees` | operation: `createEmployee`
- `team.createemployeetime` | `POST /api/team/v1/employees/{employeeId}/times` | operation: `createEmployeeTime`
- `team.delete-a-employee` | `DELETE /api/team/v1/employees/{employeeId}` | operation: `Delete a Employee`
- `team.deletetime` | `DELETE /api/team/v1/employees/times/{employeeTimeId}` | operation: `deleteTime`
- `team.employeeclockin` | `POST /api/team/v1/employees/{employeeId}/times/clockin` | operation: `employeeClockin`
- `team.employeeclockout` | `POST /api/team/v1/employees/{employeeId}/times/clockout` | operation: `employeeClockout`
- `team.employeepause` | `POST /api/team/v1/employees/{employeeId}/times/pause` | operation: `employeePause`
- `team.employeeunpause` | `POST /api/team/v1/employees/{employeeId}/times/unpause` | operation: `employeeUnpause`
- `team.get-a-employee` | `GET /api/team/v1/employees/{employeeId}` | operation: `Get a Employee`
- `team.gettime` | `GET /api/team/v1/employees/times/{employeeTimeId}` | operation: `getTime`
- `team.listemployees` | `GET /api/team/v1/employees` | operation: `listEmployees`
- `team.listemployeetimes` | `GET /api/team/v1/employees/{employeeId}/times` | operation: `listemployeeTimes`
- `team.listtimes` | `GET /api/team/v1/employees/times` | operation: `listTimes`
- `team.update-employee` | `PUT /api/team/v1/employees/{employeeId}` | operation: `Update Employee`
- `team.updatetime` | `PUT /api/team/v1/employees/times/{employeeTimeId}` | operation: `UpdateTime`

