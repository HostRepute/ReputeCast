from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.whats_app_test_result_status import WhatsAppTestResultStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="WhatsAppTestResult")



@_attrs_define
class WhatsAppTestResult:
    """ 
        Attributes:
            at (datetime.datetime):
            id (str):
            read (bool):
            status (WhatsAppTestResultStatus):
            template (str):
            to (str):
            reason (str | Unset):
     """

    at: datetime.datetime
    id: str
    read: bool
    status: WhatsAppTestResultStatus
    template: str
    to: str
    reason: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        id = self.id

        read = self.read

        status = self.status.value

        template = self.template

        to = self.to

        reason = self.reason


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "id": id,
            "read": read,
            "status": status,
            "template": template,
            "to": to,
        })
        if reason is not UNSET:
            field_dict["reason"] = reason

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        id = d.pop("id")

        read = d.pop("read")

        status = WhatsAppTestResultStatus(d.pop("status"))




        template = d.pop("template")

        to = d.pop("to")

        reason = d.pop("reason", UNSET)

        whats_app_test_result = cls(
            at=at,
            id=id,
            read=read,
            status=status,
            template=template,
            to=to,
            reason=reason,
        )

        return whats_app_test_result

