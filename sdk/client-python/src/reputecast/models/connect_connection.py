from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.connect_connection_access import ConnectConnectionAccess
from ..models.connect_connection_events import ConnectConnectionEvents
from ..models.connect_connection_events_check import ConnectConnectionEventsCheck
from ..models.connect_connection_status import ConnectConnectionStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.connect_access import ConnectAccess
  from ..models.connect_error import ConnectError
  from ..models.connect_field import ConnectField
  from ..models.connect_host import ConnectHost
  from ..models.connect_probe import ConnectProbe
  from ..models.connect_provision_row import ConnectProvisionRow
  from ..models.connect_region import ConnectRegion
  from ..models.connect_sandbox import ConnectSandbox
  from ..models.connect_smtp_form import ConnectSMTPForm
  from ..models.connect_webhook import ConnectWebhook





T = TypeVar("T", bound="ConnectConnection")



@_attrs_define
class ConnectConnection:
    """ 
        Attributes:
            access (ConnectConnectionAccess):
            access_key (str):
            account (str):
            account_id (str):
            can_continue (bool):
            events (ConnectConnectionEvents):
            events_check (ConnectConnectionEventsCheck):
            events_url (str):
            external_id (str):
            fields (list[ConnectField]):
            iam_user (str):
            provision_done (int):
            provision_total (int):
            provisioning (list[ConnectProvisionRow]):
            region (str):
            regions (list[ConnectRegion]):
            role_arn (str):
            status (ConnectConnectionStatus):
            summary (str):
            working (bool):
            at (datetime.datetime | Unset):
            error (ConnectError | Unset):
            events_at (datetime.datetime | Unset):
            host (ConnectHost | Unset):
            mailbox (ConnectProbe | Unset):
            sandbox (ConnectSandbox | Unset):
            smtp (ConnectProbe | Unset):
            smtp_form (ConnectSMTPForm | Unset):
            template (ConnectAccess | Unset):
            webhook (ConnectWebhook | Unset):
     """

    access: ConnectConnectionAccess
    access_key: str
    account: str
    account_id: str
    can_continue: bool
    events: ConnectConnectionEvents
    events_check: ConnectConnectionEventsCheck
    events_url: str
    external_id: str
    fields: list[ConnectField]
    iam_user: str
    provision_done: int
    provision_total: int
    provisioning: list[ConnectProvisionRow]
    region: str
    regions: list[ConnectRegion]
    role_arn: str
    status: ConnectConnectionStatus
    summary: str
    working: bool
    at: datetime.datetime | Unset = UNSET
    error: ConnectError | Unset = UNSET
    events_at: datetime.datetime | Unset = UNSET
    host: ConnectHost | Unset = UNSET
    mailbox: ConnectProbe | Unset = UNSET
    sandbox: ConnectSandbox | Unset = UNSET
    smtp: ConnectProbe | Unset = UNSET
    smtp_form: ConnectSMTPForm | Unset = UNSET
    template: ConnectAccess | Unset = UNSET
    webhook: ConnectWebhook | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.connect_access import ConnectAccess # noqa: PLC0415
        from ..models.connect_error import ConnectError # noqa: PLC0415
        from ..models.connect_field import ConnectField # noqa: PLC0415
        from ..models.connect_host import ConnectHost # noqa: PLC0415
        from ..models.connect_probe import ConnectProbe # noqa: PLC0415
        from ..models.connect_provision_row import ConnectProvisionRow # noqa: PLC0415
        from ..models.connect_region import ConnectRegion # noqa: PLC0415
        from ..models.connect_sandbox import ConnectSandbox # noqa: PLC0415
        from ..models.connect_smtp_form import ConnectSMTPForm # noqa: PLC0415
        from ..models.connect_webhook import ConnectWebhook # noqa: PLC0415
        access = self.access.value

        access_key = self.access_key

        account = self.account

        account_id = self.account_id

        can_continue = self.can_continue

        events = self.events.value

        events_check = self.events_check.value

        events_url = self.events_url

        external_id = self.external_id

        fields = []
        for fields_item_data in self.fields:
            fields_item = fields_item_data.to_dict()
            fields.append(fields_item)



        iam_user = self.iam_user

        provision_done = self.provision_done

        provision_total = self.provision_total

        provisioning = []
        for provisioning_item_data in self.provisioning:
            provisioning_item = provisioning_item_data.to_dict()
            provisioning.append(provisioning_item)



        region = self.region

        regions = []
        for regions_item_data in self.regions:
            regions_item = regions_item_data.to_dict()
            regions.append(regions_item)



        role_arn = self.role_arn

        status = self.status.value

        summary = self.summary

        working = self.working

        at: str | Unset = UNSET
        if not isinstance(self.at, Unset):
            at = self.at.isoformat()

        error: dict[str, Any] | Unset = UNSET
        if not isinstance(self.error, Unset):
            error = self.error.to_dict()

        events_at: str | Unset = UNSET
        if not isinstance(self.events_at, Unset):
            events_at = self.events_at.isoformat()

        host: dict[str, Any] | Unset = UNSET
        if not isinstance(self.host, Unset):
            host = self.host.to_dict()

        mailbox: dict[str, Any] | Unset = UNSET
        if not isinstance(self.mailbox, Unset):
            mailbox = self.mailbox.to_dict()

        sandbox: dict[str, Any] | Unset = UNSET
        if not isinstance(self.sandbox, Unset):
            sandbox = self.sandbox.to_dict()

        smtp: dict[str, Any] | Unset = UNSET
        if not isinstance(self.smtp, Unset):
            smtp = self.smtp.to_dict()

        smtp_form: dict[str, Any] | Unset = UNSET
        if not isinstance(self.smtp_form, Unset):
            smtp_form = self.smtp_form.to_dict()

        template: dict[str, Any] | Unset = UNSET
        if not isinstance(self.template, Unset):
            template = self.template.to_dict()

        webhook: dict[str, Any] | Unset = UNSET
        if not isinstance(self.webhook, Unset):
            webhook = self.webhook.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "access": access,
            "accessKey": access_key,
            "account": account,
            "accountId": account_id,
            "canContinue": can_continue,
            "events": events,
            "eventsCheck": events_check,
            "eventsUrl": events_url,
            "externalId": external_id,
            "fields": fields,
            "iamUser": iam_user,
            "provisionDone": provision_done,
            "provisionTotal": provision_total,
            "provisioning": provisioning,
            "region": region,
            "regions": regions,
            "roleArn": role_arn,
            "status": status,
            "summary": summary,
            "working": working,
        })
        if at is not UNSET:
            field_dict["at"] = at
        if error is not UNSET:
            field_dict["error"] = error
        if events_at is not UNSET:
            field_dict["eventsAt"] = events_at
        if host is not UNSET:
            field_dict["host"] = host
        if mailbox is not UNSET:
            field_dict["mailbox"] = mailbox
        if sandbox is not UNSET:
            field_dict["sandbox"] = sandbox
        if smtp is not UNSET:
            field_dict["smtp"] = smtp
        if smtp_form is not UNSET:
            field_dict["smtpForm"] = smtp_form
        if template is not UNSET:
            field_dict["template"] = template
        if webhook is not UNSET:
            field_dict["webhook"] = webhook

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.connect_access import ConnectAccess # noqa: PLC0415
        from ..models.connect_error import ConnectError # noqa: PLC0415
        from ..models.connect_field import ConnectField # noqa: PLC0415
        from ..models.connect_host import ConnectHost # noqa: PLC0415
        from ..models.connect_probe import ConnectProbe # noqa: PLC0415
        from ..models.connect_provision_row import ConnectProvisionRow # noqa: PLC0415
        from ..models.connect_region import ConnectRegion # noqa: PLC0415
        from ..models.connect_sandbox import ConnectSandbox # noqa: PLC0415
        from ..models.connect_smtp_form import ConnectSMTPForm # noqa: PLC0415
        from ..models.connect_webhook import ConnectWebhook # noqa: PLC0415
        d = dict(src_dict)
        access = ConnectConnectionAccess(d.pop("access"))




        access_key = d.pop("accessKey")

        account = d.pop("account")

        account_id = d.pop("accountId")

        can_continue = d.pop("canContinue")

        events = ConnectConnectionEvents(d.pop("events"))




        events_check = ConnectConnectionEventsCheck(d.pop("eventsCheck"))




        events_url = d.pop("eventsUrl")

        external_id = d.pop("externalId")

        fields = []
        _fields = d.pop("fields")
        for fields_item_data in (_fields):
            fields_item = ConnectField.from_dict(fields_item_data)



            fields.append(fields_item)


        iam_user = d.pop("iamUser")

        provision_done = d.pop("provisionDone")

        provision_total = d.pop("provisionTotal")

        provisioning = []
        _provisioning = d.pop("provisioning")
        for provisioning_item_data in (_provisioning):
            provisioning_item = ConnectProvisionRow.from_dict(provisioning_item_data)



            provisioning.append(provisioning_item)


        region = d.pop("region")

        regions = []
        _regions = d.pop("regions")
        for regions_item_data in (_regions):
            regions_item = ConnectRegion.from_dict(regions_item_data)



            regions.append(regions_item)


        role_arn = d.pop("roleArn")

        status = ConnectConnectionStatus(d.pop("status"))




        summary = d.pop("summary")

        working = d.pop("working")

        _at = d.pop("at", UNSET)
        at: datetime.datetime | Unset
        if isinstance(_at,  Unset):
            at = UNSET
        else:
            at = datetime.datetime.fromisoformat(_at)




        _error = d.pop("error", UNSET)
        error: ConnectError | Unset
        if isinstance(_error,  Unset):
            error = UNSET
        else:
            error = ConnectError.from_dict(_error)




        _events_at = d.pop("eventsAt", UNSET)
        events_at: datetime.datetime | Unset
        if isinstance(_events_at,  Unset):
            events_at = UNSET
        else:
            events_at = datetime.datetime.fromisoformat(_events_at)




        _host = d.pop("host", UNSET)
        host: ConnectHost | Unset
        if isinstance(_host,  Unset):
            host = UNSET
        else:
            host = ConnectHost.from_dict(_host)




        _mailbox = d.pop("mailbox", UNSET)
        mailbox: ConnectProbe | Unset
        if isinstance(_mailbox,  Unset):
            mailbox = UNSET
        else:
            mailbox = ConnectProbe.from_dict(_mailbox)




        _sandbox = d.pop("sandbox", UNSET)
        sandbox: ConnectSandbox | Unset
        if isinstance(_sandbox,  Unset):
            sandbox = UNSET
        else:
            sandbox = ConnectSandbox.from_dict(_sandbox)




        _smtp = d.pop("smtp", UNSET)
        smtp: ConnectProbe | Unset
        if isinstance(_smtp,  Unset):
            smtp = UNSET
        else:
            smtp = ConnectProbe.from_dict(_smtp)




        _smtp_form = d.pop("smtpForm", UNSET)
        smtp_form: ConnectSMTPForm | Unset
        if isinstance(_smtp_form,  Unset):
            smtp_form = UNSET
        else:
            smtp_form = ConnectSMTPForm.from_dict(_smtp_form)




        _template = d.pop("template", UNSET)
        template: ConnectAccess | Unset
        if isinstance(_template,  Unset):
            template = UNSET
        else:
            template = ConnectAccess.from_dict(_template)




        _webhook = d.pop("webhook", UNSET)
        webhook: ConnectWebhook | Unset
        if isinstance(_webhook,  Unset):
            webhook = UNSET
        else:
            webhook = ConnectWebhook.from_dict(_webhook)




        connect_connection = cls(
            access=access,
            access_key=access_key,
            account=account,
            account_id=account_id,
            can_continue=can_continue,
            events=events,
            events_check=events_check,
            events_url=events_url,
            external_id=external_id,
            fields=fields,
            iam_user=iam_user,
            provision_done=provision_done,
            provision_total=provision_total,
            provisioning=provisioning,
            region=region,
            regions=regions,
            role_arn=role_arn,
            status=status,
            summary=summary,
            working=working,
            at=at,
            error=error,
            events_at=events_at,
            host=host,
            mailbox=mailbox,
            sandbox=sandbox,
            smtp=smtp,
            smtp_form=smtp_form,
            template=template,
            webhook=webhook,
        )

        return connect_connection

