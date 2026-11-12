from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.connect_done_row import ConnectDoneRow





T = TypeVar("T", bound="ConnectDone")



@_attrs_define
class ConnectDone:
    """ 
        Attributes:
            domain (str):
            host_repute (bool):
            minutes (int):
            rows (list[ConnectDoneRow]):
            title (str):
     """

    domain: str
    host_repute: bool
    minutes: int
    rows: list[ConnectDoneRow]
    title: str





    def to_dict(self) -> dict[str, Any]:
        from ..models.connect_done_row import ConnectDoneRow # noqa: PLC0415
        domain = self.domain

        host_repute = self.host_repute

        minutes = self.minutes

        rows = []
        for rows_item_data in self.rows:
            rows_item = rows_item_data.to_dict()
            rows.append(rows_item)



        title = self.title


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "domain": domain,
            "hostRepute": host_repute,
            "minutes": minutes,
            "rows": rows,
            "title": title,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.connect_done_row import ConnectDoneRow # noqa: PLC0415
        d = dict(src_dict)
        domain = d.pop("domain")

        host_repute = d.pop("hostRepute")

        minutes = d.pop("minutes")

        rows = []
        _rows = d.pop("rows")
        for rows_item_data in (_rows):
            rows_item = ConnectDoneRow.from_dict(rows_item_data)



            rows.append(rows_item)


        title = d.pop("title")

        connect_done = cls(
            domain=domain,
            host_repute=host_repute,
            minutes=minutes,
            rows=rows,
            title=title,
        )

        return connect_done

