from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ConnectAccess")



@_attrs_define
class ConnectAccess:
    """ 
        Attributes:
            launch_url (str):
            policy (str):
            principal (str):
            role_name (str):
     """

    launch_url: str
    policy: str
    principal: str
    role_name: str





    def to_dict(self) -> dict[str, Any]:
        launch_url = self.launch_url

        policy = self.policy

        principal = self.principal

        role_name = self.role_name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "launchUrl": launch_url,
            "policy": policy,
            "principal": principal,
            "roleName": role_name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        launch_url = d.pop("launchUrl")

        policy = d.pop("policy")

        principal = d.pop("principal")

        role_name = d.pop("roleName")

        connect_access = cls(
            launch_url=launch_url,
            policy=policy,
            principal=principal,
            role_name=role_name,
        )

        return connect_access

