from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="PermissionInfo")



@_attrs_define
class PermissionInfo:
    """ 
        Attributes:
            description (str):
            group (str):  Example: Campaigns.
            implies (list[str]):
            key (str):  Example: campaigns:send.
            label (str):  Example: Send or schedule campaigns.
            no_agents (bool):
            owner_only (bool):
            people_only (bool):
            sensitive (bool):
     """

    description: str
    group: str
    implies: list[str]
    key: str
    label: str
    no_agents: bool
    owner_only: bool
    people_only: bool
    sensitive: bool





    def to_dict(self) -> dict[str, Any]:
        description = self.description

        group = self.group

        implies = self.implies



        key = self.key

        label = self.label

        no_agents = self.no_agents

        owner_only = self.owner_only

        people_only = self.people_only

        sensitive = self.sensitive


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "description": description,
            "group": group,
            "implies": implies,
            "key": key,
            "label": label,
            "noAgents": no_agents,
            "ownerOnly": owner_only,
            "peopleOnly": people_only,
            "sensitive": sensitive,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        description = d.pop("description")

        group = d.pop("group")

        implies = cast(list[str], d.pop("implies"))


        key = d.pop("key")

        label = d.pop("label")

        no_agents = d.pop("noAgents")

        owner_only = d.pop("ownerOnly")

        people_only = d.pop("peopleOnly")

        sensitive = d.pop("sensitive")

        permission_info = cls(
            description=description,
            group=group,
            implies=implies,
            key=key,
            label=label,
            no_agents=no_agents,
            owner_only=owner_only,
            people_only=people_only,
            sensitive=sensitive,
        )

        return permission_info

