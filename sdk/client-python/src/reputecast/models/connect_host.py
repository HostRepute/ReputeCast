from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.connect_host_recommended import ConnectHostRecommended
from typing import cast
import datetime






T = TypeVar("T", bound="ConnectHost")



@_attrs_define
class ConnectHost:
    """ 
        Attributes:
            base_credentials (bool):
            checked_at (datetime.datetime):
            kind (str):
            on_aws (bool):
            recommended (ConnectHostRecommended):
     """

    base_credentials: bool
    checked_at: datetime.datetime
    kind: str
    on_aws: bool
    recommended: ConnectHostRecommended





    def to_dict(self) -> dict[str, Any]:
        base_credentials = self.base_credentials

        checked_at = self.checked_at.isoformat()

        kind = self.kind

        on_aws = self.on_aws

        recommended = self.recommended.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "baseCredentials": base_credentials,
            "checkedAt": checked_at,
            "kind": kind,
            "onAws": on_aws,
            "recommended": recommended,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        base_credentials = d.pop("baseCredentials")

        checked_at = datetime.datetime.fromisoformat(d.pop("checkedAt"))




        kind = d.pop("kind")

        on_aws = d.pop("onAws")

        recommended = ConnectHostRecommended(d.pop("recommended"))




        connect_host = cls(
            base_credentials=base_credentials,
            checked_at=checked_at,
            kind=kind,
            on_aws=on_aws,
            recommended=recommended,
        )

        return connect_host

