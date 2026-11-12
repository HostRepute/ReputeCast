from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.connect_stream_view_role import ConnectStreamViewRole
from ..models.connect_stream_view_stream import ConnectStreamViewStream
from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="ConnectStreamView")



@_attrs_define
class ConnectStreamView:
    """ 
        Attributes:
            allowed (bool):
            configuration_set (str):
            current (list[str]):
            description (str):
            enabled (bool):
            failover (str):
            has_route (bool):
            name (str):
            role (ConnectStreamViewRole):
            role_label (str):
            stream (ConnectStreamViewStream):
            tracking (str):
            weight (int):
            reason (str | Unset):
     """

    allowed: bool
    configuration_set: str
    current: list[str]
    description: str
    enabled: bool
    failover: str
    has_route: bool
    name: str
    role: ConnectStreamViewRole
    role_label: str
    stream: ConnectStreamViewStream
    tracking: str
    weight: int
    reason: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        allowed = self.allowed

        configuration_set = self.configuration_set

        current = self.current



        description = self.description

        enabled = self.enabled

        failover = self.failover

        has_route = self.has_route

        name = self.name

        role = self.role.value

        role_label = self.role_label

        stream = self.stream.value

        tracking = self.tracking

        weight = self.weight

        reason = self.reason


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "allowed": allowed,
            "configurationSet": configuration_set,
            "current": current,
            "description": description,
            "enabled": enabled,
            "failover": failover,
            "hasRoute": has_route,
            "name": name,
            "role": role,
            "roleLabel": role_label,
            "stream": stream,
            "tracking": tracking,
            "weight": weight,
        })
        if reason is not UNSET:
            field_dict["reason"] = reason

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        allowed = d.pop("allowed")

        configuration_set = d.pop("configurationSet")

        current = cast(list[str], d.pop("current"))


        description = d.pop("description")

        enabled = d.pop("enabled")

        failover = d.pop("failover")

        has_route = d.pop("hasRoute")

        name = d.pop("name")

        role = ConnectStreamViewRole(d.pop("role"))




        role_label = d.pop("roleLabel")

        stream = ConnectStreamViewStream(d.pop("stream"))




        tracking = d.pop("tracking")

        weight = d.pop("weight")

        reason = d.pop("reason", UNSET)

        connect_stream_view = cls(
            allowed=allowed,
            configuration_set=configuration_set,
            current=current,
            description=description,
            enabled=enabled,
            failover=failover,
            has_route=has_route,
            name=name,
            role=role,
            role_label=role_label,
            stream=stream,
            tracking=tracking,
            weight=weight,
            reason=reason,
        )

        return connect_stream_view

