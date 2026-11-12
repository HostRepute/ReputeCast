from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="TeamEffectivePermission")



@_attrs_define
class TeamEffectivePermission:
    """ 
        Attributes:
            granted (bool):
            group (str):
            key (str):
            label (str):
            scope (str):
            sensitive (bool):
            source (str):
            included_by (str | Unset):
     """

    granted: bool
    group: str
    key: str
    label: str
    scope: str
    sensitive: bool
    source: str
    included_by: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        granted = self.granted

        group = self.group

        key = self.key

        label = self.label

        scope = self.scope

        sensitive = self.sensitive

        source = self.source

        included_by = self.included_by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "granted": granted,
            "group": group,
            "key": key,
            "label": label,
            "scope": scope,
            "sensitive": sensitive,
            "source": source,
        })
        if included_by is not UNSET:
            field_dict["includedBy"] = included_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        granted = d.pop("granted")

        group = d.pop("group")

        key = d.pop("key")

        label = d.pop("label")

        scope = d.pop("scope")

        sensitive = d.pop("sensitive")

        source = d.pop("source")

        included_by = d.pop("includedBy", UNSET)

        team_effective_permission = cls(
            granted=granted,
            group=group,
            key=key,
            label=label,
            scope=scope,
            sensitive=sensitive,
            source=source,
            included_by=included_by,
        )

        return team_effective_permission

