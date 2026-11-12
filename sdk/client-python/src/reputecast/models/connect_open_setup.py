from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="ConnectOpenSetup")



@_attrs_define
class ConnectOpenSetup:
    """ 
        Attributes:
            adapter (str):
            id (str):
            name (str):
            step (str):
            updated_at (datetime.datetime):
     """

    adapter: str
    id: str
    name: str
    step: str
    updated_at: datetime.datetime





    def to_dict(self) -> dict[str, Any]:
        adapter = self.adapter

        id = self.id

        name = self.name

        step = self.step

        updated_at = self.updated_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "adapter": adapter,
            "id": id,
            "name": name,
            "step": step,
            "updatedAt": updated_at,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        adapter = d.pop("adapter")

        id = d.pop("id")

        name = d.pop("name")

        step = d.pop("step")

        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))




        connect_open_setup = cls(
            adapter=adapter,
            id=id,
            name=name,
            step=step,
            updated_at=updated_at,
        )

        return connect_open_setup

