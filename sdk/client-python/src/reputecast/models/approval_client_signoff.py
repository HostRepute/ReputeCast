from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ApprovalClientSignoff")



@_attrs_define
class ApprovalClientSignoff:
    """ 
        Attributes:
            automations (bool):
            enabled (bool):
     """

    automations: bool
    enabled: bool





    def to_dict(self) -> dict[str, Any]:
        automations = self.automations

        enabled = self.enabled


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "automations": automations,
            "enabled": enabled,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        automations = d.pop("automations")

        enabled = d.pop("enabled")

        approval_client_signoff = cls(
            automations=automations,
            enabled=enabled,
        )

        return approval_client_signoff

