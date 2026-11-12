from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.automation_email_step_over_cap import AutomationEmailStepOverCap
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.automation_send_window import AutomationSendWindow





T = TypeVar("T", bound="AutomationEmailStep")



@_attrs_define
class AutomationEmailStep:
    """ 
        Attributes:
            email_id (str | Unset):
            email_version (int | Unset):
            over_cap (AutomationEmailStepOverCap | Unset):
            route (str | Unset):
            send_window (AutomationSendWindow | Unset):
            skip_list_id (str | Unset):
            skip_recent_hours (int | Unset):
            subject (str | Unset):
     """

    email_id: str | Unset = UNSET
    email_version: int | Unset = UNSET
    over_cap: AutomationEmailStepOverCap | Unset = UNSET
    route: str | Unset = UNSET
    send_window: AutomationSendWindow | Unset = UNSET
    skip_list_id: str | Unset = UNSET
    skip_recent_hours: int | Unset = UNSET
    subject: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.automation_send_window import AutomationSendWindow # noqa: PLC0415
        email_id = self.email_id

        email_version = self.email_version

        over_cap: str | Unset = UNSET
        if not isinstance(self.over_cap, Unset):
            over_cap = self.over_cap.value


        route = self.route

        send_window: dict[str, Any] | Unset = UNSET
        if not isinstance(self.send_window, Unset):
            send_window = self.send_window.to_dict()

        skip_list_id = self.skip_list_id

        skip_recent_hours = self.skip_recent_hours

        subject = self.subject


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if email_id is not UNSET:
            field_dict["emailId"] = email_id
        if email_version is not UNSET:
            field_dict["emailVersion"] = email_version
        if over_cap is not UNSET:
            field_dict["overCap"] = over_cap
        if route is not UNSET:
            field_dict["route"] = route
        if send_window is not UNSET:
            field_dict["sendWindow"] = send_window
        if skip_list_id is not UNSET:
            field_dict["skipListId"] = skip_list_id
        if skip_recent_hours is not UNSET:
            field_dict["skipRecentHours"] = skip_recent_hours
        if subject is not UNSET:
            field_dict["subject"] = subject

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.automation_send_window import AutomationSendWindow # noqa: PLC0415
        d = dict(src_dict)
        email_id = d.pop("emailId", UNSET)

        email_version = d.pop("emailVersion", UNSET)

        _over_cap = d.pop("overCap", UNSET)
        over_cap: AutomationEmailStepOverCap | Unset
        if isinstance(_over_cap,  Unset):
            over_cap = UNSET
        else:
            over_cap = AutomationEmailStepOverCap(_over_cap)




        route = d.pop("route", UNSET)

        _send_window = d.pop("sendWindow", UNSET)
        send_window: AutomationSendWindow | Unset
        if isinstance(_send_window,  Unset):
            send_window = UNSET
        else:
            send_window = AutomationSendWindow.from_dict(_send_window)




        skip_list_id = d.pop("skipListId", UNSET)

        skip_recent_hours = d.pop("skipRecentHours", UNSET)

        subject = d.pop("subject", UNSET)

        automation_email_step = cls(
            email_id=email_id,
            email_version=email_version,
            over_cap=over_cap,
            route=route,
            send_window=send_window,
            skip_list_id=skip_list_id,
            skip_recent_hours=skip_recent_hours,
            subject=subject,
        )

        return automation_email_step

