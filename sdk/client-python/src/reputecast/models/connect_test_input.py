from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
from uuid import UUID

if TYPE_CHECKING:
  from ..models.connect_test_input_seeds import ConnectTestInputSeeds





T = TypeVar("T", bound="ConnectTestInput")



@_attrs_define
class ConnectTestInput:
    """ 
        Attributes:
            sender_id (UUID):
            subject (str):
            seeds (ConnectTestInputSeeds | Unset):
            to (str | Unset):
     """

    sender_id: UUID
    subject: str
    seeds: ConnectTestInputSeeds | Unset = UNSET
    to: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.connect_test_input_seeds import ConnectTestInputSeeds # noqa: PLC0415
        sender_id = str(self.sender_id)

        subject = self.subject

        seeds: dict[str, Any] | Unset = UNSET
        if not isinstance(self.seeds, Unset):
            seeds = self.seeds.to_dict()

        to = self.to


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "senderId": sender_id,
            "subject": subject,
        })
        if seeds is not UNSET:
            field_dict["seeds"] = seeds
        if to is not UNSET:
            field_dict["to"] = to

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.connect_test_input_seeds import ConnectTestInputSeeds # noqa: PLC0415
        d = dict(src_dict)
        sender_id = UUID(d.pop("senderId"))




        subject = d.pop("subject")

        _seeds = d.pop("seeds", UNSET)
        seeds: ConnectTestInputSeeds | Unset
        if isinstance(_seeds,  Unset):
            seeds = UNSET
        else:
            seeds = ConnectTestInputSeeds.from_dict(_seeds)




        to = d.pop("to", UNSET)

        connect_test_input = cls(
            sender_id=sender_id,
            subject=subject,
            seeds=seeds,
            to=to,
        )

        return connect_test_input

