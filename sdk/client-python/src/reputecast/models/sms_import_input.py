from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.sms_import_sender import SmsImportSender





T = TypeVar("T", bound="SmsImportInput")



@_attrs_define
class SmsImportInput:
    """ 
        Attributes:
            senders (list[SmsImportSender]):
            setup_id (str | Unset):
     """

    senders: list[SmsImportSender]
    setup_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.sms_import_sender import SmsImportSender # noqa: PLC0415
        senders = []
        for senders_item_data in self.senders:
            senders_item = senders_item_data.to_dict()
            senders.append(senders_item)



        setup_id = self.setup_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "senders": senders,
        })
        if setup_id is not UNSET:
            field_dict["setupId"] = setup_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sms_import_sender import SmsImportSender # noqa: PLC0415
        d = dict(src_dict)
        senders = []
        _senders = d.pop("senders")
        for senders_item_data in (_senders):
            senders_item = SmsImportSender.from_dict(senders_item_data)



            senders.append(senders_item)


        setup_id = d.pop("setupId", UNSET)

        sms_import_input = cls(
            senders=senders,
            setup_id=setup_id,
        )

        return sms_import_input

