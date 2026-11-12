from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="IntegrationOwned")



@_attrs_define
class IntegrationOwned:
    """ 
        Attributes:
            connection_id (str):
            key_name (str):
            kind (str):
            label (str):
            workflows (int): Workflows seen in the last 30 days
            silent_since (datetime.datetime | Unset):
     """

    connection_id: str
    key_name: str
    kind: str
    label: str
    workflows: int
    silent_since: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        connection_id = self.connection_id

        key_name = self.key_name

        kind = self.kind

        label = self.label

        workflows = self.workflows

        silent_since: str | Unset = UNSET
        if not isinstance(self.silent_since, Unset):
            silent_since = self.silent_since.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "connectionId": connection_id,
            "keyName": key_name,
            "kind": kind,
            "label": label,
            "workflows": workflows,
        })
        if silent_since is not UNSET:
            field_dict["silentSince"] = silent_since

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        connection_id = d.pop("connectionId")

        key_name = d.pop("keyName")

        kind = d.pop("kind")

        label = d.pop("label")

        workflows = d.pop("workflows")

        _silent_since = d.pop("silentSince", UNSET)
        silent_since: datetime.datetime | Unset
        if isinstance(_silent_since,  Unset):
            silent_since = UNSET
        else:
            silent_since = datetime.datetime.fromisoformat(_silent_since)




        integration_owned = cls(
            connection_id=connection_id,
            key_name=key_name,
            kind=kind,
            label=label,
            workflows=workflows,
            silent_since=silent_since,
        )

        return integration_owned

