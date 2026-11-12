from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sms_suppression_reason import SmsSuppressionReason
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="SmsSuppression")



@_attrs_define
class SmsSuppression:
    """ 
        Attributes:
            at (datetime.datetime):
            code (str):
            id (str):
            number (str):
            reason (SmsSuppressionReason):
            removable (bool):
            source (str):
            automation (str | Unset):
            campaign (str | Unset):
            context (str | Unset):
     """

    at: datetime.datetime
    code: str
    id: str
    number: str
    reason: SmsSuppressionReason
    removable: bool
    source: str
    automation: str | Unset = UNSET
    campaign: str | Unset = UNSET
    context: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        code = self.code

        id = self.id

        number = self.number

        reason = self.reason.value

        removable = self.removable

        source = self.source

        automation = self.automation

        campaign = self.campaign

        context = self.context


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "code": code,
            "id": id,
            "number": number,
            "reason": reason,
            "removable": removable,
            "source": source,
        })
        if automation is not UNSET:
            field_dict["automation"] = automation
        if campaign is not UNSET:
            field_dict["campaign"] = campaign
        if context is not UNSET:
            field_dict["context"] = context

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        code = d.pop("code")

        id = d.pop("id")

        number = d.pop("number")

        reason = SmsSuppressionReason(d.pop("reason"))




        removable = d.pop("removable")

        source = d.pop("source")

        automation = d.pop("automation", UNSET)

        campaign = d.pop("campaign", UNSET)

        context = d.pop("context", UNSET)

        sms_suppression = cls(
            at=at,
            code=code,
            id=id,
            number=number,
            reason=reason,
            removable=removable,
            source=source,
            automation=automation,
            campaign=campaign,
            context=context,
        )

        return sms_suppression

