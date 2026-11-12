from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.connect_provision_row_status import ConnectProvisionRowStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ConnectProvisionRow")



@_attrs_define
class ConnectProvisionRow:
    """ 
        Attributes:
            key (str):
            status (ConnectProvisionRowStatus):
            text (str):
            at (datetime.datetime | Unset):
            error (str | Unset):
            hint (str | Unset):
            note (str | Unset):
     """

    key: str
    status: ConnectProvisionRowStatus
    text: str
    at: datetime.datetime | Unset = UNSET
    error: str | Unset = UNSET
    hint: str | Unset = UNSET
    note: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        key = self.key

        status = self.status.value

        text = self.text

        at: str | Unset = UNSET
        if not isinstance(self.at, Unset):
            at = self.at.isoformat()

        error = self.error

        hint = self.hint

        note = self.note


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "key": key,
            "status": status,
            "text": text,
        })
        if at is not UNSET:
            field_dict["at"] = at
        if error is not UNSET:
            field_dict["error"] = error
        if hint is not UNSET:
            field_dict["hint"] = hint
        if note is not UNSET:
            field_dict["note"] = note

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        key = d.pop("key")

        status = ConnectProvisionRowStatus(d.pop("status"))




        text = d.pop("text")

        _at = d.pop("at", UNSET)
        at: datetime.datetime | Unset
        if isinstance(_at,  Unset):
            at = UNSET
        else:
            at = datetime.datetime.fromisoformat(_at)




        error = d.pop("error", UNSET)

        hint = d.pop("hint", UNSET)

        note = d.pop("note", UNSET)

        connect_provision_row = cls(
            key=key,
            status=status,
            text=text,
            at=at,
            error=error,
            hint=hint,
            note=note,
        )

        return connect_provision_row

