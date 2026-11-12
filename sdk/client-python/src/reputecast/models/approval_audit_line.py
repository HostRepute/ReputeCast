from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ApprovalAuditLine")



@_attrs_define
class ApprovalAuditLine:
    """ 
        Attributes:
            action (str):
            actor (str):
            at (datetime.datetime):
            ip (str | Unset):
     """

    action: str
    actor: str
    at: datetime.datetime
    ip: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        action = self.action

        actor = self.actor

        at = self.at.isoformat()

        ip = self.ip


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "action": action,
            "actor": actor,
            "at": at,
        })
        if ip is not UNSET:
            field_dict["ip"] = ip

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        action = d.pop("action")

        actor = d.pop("actor")

        at = datetime.datetime.fromisoformat(d.pop("at"))




        ip = d.pop("ip", UNSET)

        approval_audit_line = cls(
            action=action,
            actor=actor,
            at=at,
            ip=ip,
        )

        return approval_audit_line

