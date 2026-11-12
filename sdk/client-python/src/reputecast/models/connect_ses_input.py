from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.connect_ses_input_access import ConnectSESInputAccess
from ..models.connect_ses_input_events import ConnectSESInputEvents
from ..types import UNSET, Unset






T = TypeVar("T", bound="ConnectSESInput")



@_attrs_define
class ConnectSESInput:
    """ 
        Attributes:
            access (ConnectSESInputAccess):
            access_key_id (str | Unset):
            events (ConnectSESInputEvents | Unset):
            region (str | Unset):
            role_arn (str | Unset):
            secret_access_key (str | Unset):
     """

    access: ConnectSESInputAccess
    access_key_id: str | Unset = UNSET
    events: ConnectSESInputEvents | Unset = UNSET
    region: str | Unset = UNSET
    role_arn: str | Unset = UNSET
    secret_access_key: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        access = self.access.value

        access_key_id = self.access_key_id

        events: str | Unset = UNSET
        if not isinstance(self.events, Unset):
            events = self.events.value


        region = self.region

        role_arn = self.role_arn

        secret_access_key = self.secret_access_key


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "access": access,
        })
        if access_key_id is not UNSET:
            field_dict["accessKeyId"] = access_key_id
        if events is not UNSET:
            field_dict["events"] = events
        if region is not UNSET:
            field_dict["region"] = region
        if role_arn is not UNSET:
            field_dict["roleArn"] = role_arn
        if secret_access_key is not UNSET:
            field_dict["secretAccessKey"] = secret_access_key

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        access = ConnectSESInputAccess(d.pop("access"))




        access_key_id = d.pop("accessKeyId", UNSET)

        _events = d.pop("events", UNSET)
        events: ConnectSESInputEvents | Unset
        if isinstance(_events,  Unset):
            events = UNSET
        else:
            events = ConnectSESInputEvents(_events)




        region = d.pop("region", UNSET)

        role_arn = d.pop("roleArn", UNSET)

        secret_access_key = d.pop("secretAccessKey", UNSET)

        connect_ses_input = cls(
            access=access,
            access_key_id=access_key_id,
            events=events,
            region=region,
            role_arn=role_arn,
            secret_access_key=secret_access_key,
        )

        return connect_ses_input

