from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="LiveWarmupTarget")



@_attrs_define
class LiveWarmupTarget:
    """ 
        Attributes:
            allowed (bool):
            name (str):
            provider_id (str):
            short (str):
            reason (str | Unset):
     """

    allowed: bool
    name: str
    provider_id: str
    short: str
    reason: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        allowed = self.allowed

        name = self.name

        provider_id = self.provider_id

        short = self.short

        reason = self.reason


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "allowed": allowed,
            "name": name,
            "providerId": provider_id,
            "short": short,
        })
        if reason is not UNSET:
            field_dict["reason"] = reason

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        allowed = d.pop("allowed")

        name = d.pop("name")

        provider_id = d.pop("providerId")

        short = d.pop("short")

        reason = d.pop("reason", UNSET)

        live_warmup_target = cls(
            allowed=allowed,
            name=name,
            provider_id=provider_id,
            short=short,
            reason=reason,
        )

        return live_warmup_target

