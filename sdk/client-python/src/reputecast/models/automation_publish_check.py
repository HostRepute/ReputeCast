from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.automation_publish_check_status import AutomationPublishCheckStatus






T = TypeVar("T", bound="AutomationPublishCheck")



@_attrs_define
class AutomationPublishCheck:
    """ 
        Attributes:
            label (str):
            status (AutomationPublishCheckStatus):
     """

    label: str
    status: AutomationPublishCheckStatus





    def to_dict(self) -> dict[str, Any]:
        label = self.label

        status = self.status.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "label": label,
            "status": status,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        label = d.pop("label")

        status = AutomationPublishCheckStatus(d.pop("status"))




        automation_publish_check = cls(
            label=label,
            status=status,
        )

        return automation_publish_check

