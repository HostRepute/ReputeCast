from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="ConnectCommunityEntry")



@_attrs_define
class ConnectCommunityEntry:
    """ 
        Attributes:
            adapter (str):
            configure (bool):
            id (str):
            maintainer (str):
            name (str):
            summary (str):
            version (str):
            reason (str | Unset):
     """

    adapter: str
    configure: bool
    id: str
    maintainer: str
    name: str
    summary: str
    version: str
    reason: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        adapter = self.adapter

        configure = self.configure

        id = self.id

        maintainer = self.maintainer

        name = self.name

        summary = self.summary

        version = self.version

        reason = self.reason


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "adapter": adapter,
            "configure": configure,
            "id": id,
            "maintainer": maintainer,
            "name": name,
            "summary": summary,
            "version": version,
        })
        if reason is not UNSET:
            field_dict["reason"] = reason

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        adapter = d.pop("adapter")

        configure = d.pop("configure")

        id = d.pop("id")

        maintainer = d.pop("maintainer")

        name = d.pop("name")

        summary = d.pop("summary")

        version = d.pop("version")

        reason = d.pop("reason", UNSET)

        connect_community_entry = cls(
            adapter=adapter,
            configure=configure,
            id=id,
            maintainer=maintainer,
            name=name,
            summary=summary,
            version=version,
            reason=reason,
        )

        return connect_community_entry

