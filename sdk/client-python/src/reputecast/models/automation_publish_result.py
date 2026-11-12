from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.automation_publish_result_status import AutomationPublishResultStatus
from ..types import UNSET, Unset






T = TypeVar("T", bound="AutomationPublishResult")



@_attrs_define
class AutomationPublishResult:
    """ 
        Attributes:
            status (AutomationPublishResultStatus):
            version (int):
            approval_id (str | Unset):
            message (str | Unset):
     """

    status: AutomationPublishResultStatus
    version: int
    approval_id: str | Unset = UNSET
    message: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        status = self.status.value

        version = self.version

        approval_id = self.approval_id

        message = self.message


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "status": status,
            "version": version,
        })
        if approval_id is not UNSET:
            field_dict["approvalId"] = approval_id
        if message is not UNSET:
            field_dict["message"] = message

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        status = AutomationPublishResultStatus(d.pop("status"))




        version = d.pop("version")

        approval_id = d.pop("approvalId", UNSET)

        message = d.pop("message", UNSET)

        automation_publish_result = cls(
            status=status,
            version=version,
            approval_id=approval_id,
            message=message,
        )

        return automation_publish_result

