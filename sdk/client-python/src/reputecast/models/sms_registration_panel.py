from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.sms_page_check import SmsPageCheck
  from ..models.sms_registration_helper import SmsRegistrationHelper
  from ..models.sms_registration_panel_details import SmsRegistrationPanelDetails
  from ..models.sms_registration_view import SmsRegistrationView
  from ..models.sms_sender_row import SmsSenderRow





T = TypeVar("T", bound="SmsRegistrationPanel")



@_attrs_define
class SmsRegistrationPanel:
    """ 
        Attributes:
            can_manage (bool):
            checks (list[SmsPageCheck]):
            console (str):
            details (SmsRegistrationPanelDetails):
            held (int):
            helper (SmsRegistrationHelper):
            items (list[SmsRegistrationView]):
            read_from (bool):
            sender (SmsSenderRow):
            read_at (datetime.datetime | Unset):
     """

    can_manage: bool
    checks: list[SmsPageCheck]
    console: str
    details: SmsRegistrationPanelDetails
    held: int
    helper: SmsRegistrationHelper
    items: list[SmsRegistrationView]
    read_from: bool
    sender: SmsSenderRow
    read_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.sms_page_check import SmsPageCheck # noqa: PLC0415
        from ..models.sms_registration_helper import SmsRegistrationHelper # noqa: PLC0415
        from ..models.sms_registration_panel_details import SmsRegistrationPanelDetails # noqa: PLC0415
        from ..models.sms_registration_view import SmsRegistrationView # noqa: PLC0415
        from ..models.sms_sender_row import SmsSenderRow # noqa: PLC0415
        can_manage = self.can_manage

        checks = []
        for checks_item_data in self.checks:
            checks_item = checks_item_data.to_dict()
            checks.append(checks_item)



        console = self.console

        details = self.details.to_dict()

        held = self.held

        helper = self.helper.to_dict()

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        read_from = self.read_from

        sender = self.sender.to_dict()

        read_at: str | Unset = UNSET
        if not isinstance(self.read_at, Unset):
            read_at = self.read_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canManage": can_manage,
            "checks": checks,
            "console": console,
            "details": details,
            "held": held,
            "helper": helper,
            "items": items,
            "readFrom": read_from,
            "sender": sender,
        })
        if read_at is not UNSET:
            field_dict["readAt"] = read_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sms_page_check import SmsPageCheck # noqa: PLC0415
        from ..models.sms_registration_helper import SmsRegistrationHelper # noqa: PLC0415
        from ..models.sms_registration_panel_details import SmsRegistrationPanelDetails # noqa: PLC0415
        from ..models.sms_registration_view import SmsRegistrationView # noqa: PLC0415
        from ..models.sms_sender_row import SmsSenderRow # noqa: PLC0415
        d = dict(src_dict)
        can_manage = d.pop("canManage")

        checks = []
        _checks = d.pop("checks")
        for checks_item_data in (_checks):
            checks_item = SmsPageCheck.from_dict(checks_item_data)



            checks.append(checks_item)


        console = d.pop("console")

        details = SmsRegistrationPanelDetails.from_dict(d.pop("details"))




        held = d.pop("held")

        helper = SmsRegistrationHelper.from_dict(d.pop("helper"))




        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = SmsRegistrationView.from_dict(items_item_data)



            items.append(items_item)


        read_from = d.pop("readFrom")

        sender = SmsSenderRow.from_dict(d.pop("sender"))




        _read_at = d.pop("readAt", UNSET)
        read_at: datetime.datetime | Unset
        if isinstance(_read_at,  Unset):
            read_at = UNSET
        else:
            read_at = datetime.datetime.fromisoformat(_read_at)




        sms_registration_panel = cls(
            can_manage=can_manage,
            checks=checks,
            console=console,
            details=details,
            held=held,
            helper=helper,
            items=items,
            read_from=read_from,
            sender=sender,
            read_at=read_at,
        )

        return sms_registration_panel

