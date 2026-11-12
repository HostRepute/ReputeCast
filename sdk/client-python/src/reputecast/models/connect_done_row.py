from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.connect_done_row_status import ConnectDoneRowStatus






T = TypeVar("T", bound="ConnectDoneRow")



@_attrs_define
class ConnectDoneRow:
    """ 
        Attributes:
            import_link (bool):
            lead (str):
            status (ConnectDoneRowStatus):
            text (str):
     """

    import_link: bool
    lead: str
    status: ConnectDoneRowStatus
    text: str





    def to_dict(self) -> dict[str, Any]:
        import_link = self.import_link

        lead = self.lead

        status = self.status.value

        text = self.text


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "importLink": import_link,
            "lead": lead,
            "status": status,
            "text": text,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        import_link = d.pop("importLink")

        lead = d.pop("lead")

        status = ConnectDoneRowStatus(d.pop("status"))




        text = d.pop("text")

        connect_done_row = cls(
            import_link=import_link,
            lead=lead,
            status=status,
            text=text,
        )

        return connect_done_row

