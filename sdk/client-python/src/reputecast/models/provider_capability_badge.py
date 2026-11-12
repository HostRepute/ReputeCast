from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.provider_capability_badge_kind import ProviderCapabilityBadgeKind






T = TypeVar("T", bound="ProviderCapabilityBadge")



@_attrs_define
class ProviderCapabilityBadge:
    """ 
        Attributes:
            kind (ProviderCapabilityBadgeKind):
            label (str):
     """

    kind: ProviderCapabilityBadgeKind
    label: str





    def to_dict(self) -> dict[str, Any]:
        kind = self.kind.value

        label = self.label


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
            "label": label,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kind = ProviderCapabilityBadgeKind(d.pop("kind"))




        label = d.pop("label")

        provider_capability_badge = cls(
            kind=kind,
            label=label,
        )

        return provider_capability_badge

