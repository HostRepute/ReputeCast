from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.connect_adapter_info_flow import ConnectAdapterInfoFlow
from typing import cast

if TYPE_CHECKING:
  from ..models.connect_field import ConnectField





T = TypeVar("T", bound="ConnectAdapterInfo")



@_attrs_define
class ConnectAdapterInfo:
    """ 
        Attributes:
            adapter (str):
            bulk_allowed (bool):
            can_poll (bool):
            can_register (bool):
            connection_hint (str):
            domain_hint (str):
            fields (list[ConnectField]):
            flow (ConnectAdapterInfoFlow):
            heading (str):
            key_hint (str):
            keys_url (str):
            lead (str):
            name (str):
            section (str):
            short (str):
            step1 (str):
            title (str):
            verify_call (str):
            webhook_events (str):
            webhooks (str):
     """

    adapter: str
    bulk_allowed: bool
    can_poll: bool
    can_register: bool
    connection_hint: str
    domain_hint: str
    fields: list[ConnectField]
    flow: ConnectAdapterInfoFlow
    heading: str
    key_hint: str
    keys_url: str
    lead: str
    name: str
    section: str
    short: str
    step1: str
    title: str
    verify_call: str
    webhook_events: str
    webhooks: str





    def to_dict(self) -> dict[str, Any]:
        from ..models.connect_field import ConnectField # noqa: PLC0415
        adapter = self.adapter

        bulk_allowed = self.bulk_allowed

        can_poll = self.can_poll

        can_register = self.can_register

        connection_hint = self.connection_hint

        domain_hint = self.domain_hint

        fields = []
        for fields_item_data in self.fields:
            fields_item = fields_item_data.to_dict()
            fields.append(fields_item)



        flow = self.flow.value

        heading = self.heading

        key_hint = self.key_hint

        keys_url = self.keys_url

        lead = self.lead

        name = self.name

        section = self.section

        short = self.short

        step1 = self.step1

        title = self.title

        verify_call = self.verify_call

        webhook_events = self.webhook_events

        webhooks = self.webhooks


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "adapter": adapter,
            "bulkAllowed": bulk_allowed,
            "canPoll": can_poll,
            "canRegister": can_register,
            "connectionHint": connection_hint,
            "domainHint": domain_hint,
            "fields": fields,
            "flow": flow,
            "heading": heading,
            "keyHint": key_hint,
            "keysUrl": keys_url,
            "lead": lead,
            "name": name,
            "section": section,
            "short": short,
            "step1": step1,
            "title": title,
            "verifyCall": verify_call,
            "webhookEvents": webhook_events,
            "webhooks": webhooks,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.connect_field import ConnectField # noqa: PLC0415
        d = dict(src_dict)
        adapter = d.pop("adapter")

        bulk_allowed = d.pop("bulkAllowed")

        can_poll = d.pop("canPoll")

        can_register = d.pop("canRegister")

        connection_hint = d.pop("connectionHint")

        domain_hint = d.pop("domainHint")

        fields = []
        _fields = d.pop("fields")
        for fields_item_data in (_fields):
            fields_item = ConnectField.from_dict(fields_item_data)



            fields.append(fields_item)


        flow = ConnectAdapterInfoFlow(d.pop("flow"))




        heading = d.pop("heading")

        key_hint = d.pop("keyHint")

        keys_url = d.pop("keysUrl")

        lead = d.pop("lead")

        name = d.pop("name")

        section = d.pop("section")

        short = d.pop("short")

        step1 = d.pop("step1")

        title = d.pop("title")

        verify_call = d.pop("verifyCall")

        webhook_events = d.pop("webhookEvents")

        webhooks = d.pop("webhooks")

        connect_adapter_info = cls(
            adapter=adapter,
            bulk_allowed=bulk_allowed,
            can_poll=can_poll,
            can_register=can_register,
            connection_hint=connection_hint,
            domain_hint=domain_hint,
            fields=fields,
            flow=flow,
            heading=heading,
            key_hint=key_hint,
            keys_url=keys_url,
            lead=lead,
            name=name,
            section=section,
            short=short,
            step1=step1,
            title=title,
            verify_call=verify_call,
            webhook_events=webhook_events,
            webhooks=webhooks,
        )

        return connect_adapter_info

