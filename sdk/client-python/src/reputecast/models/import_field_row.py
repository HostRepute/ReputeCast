from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.import_field_row_status import ImportFieldRowStatus
from typing import cast

if TYPE_CHECKING:
  from ..models.import_field_option import ImportFieldOption





T = TypeVar("T", bound="ImportFieldRow")



@_attrs_define
class ImportFieldRow:
    """ 
        Attributes:
            label (str):
            options (list[ImportFieldOption]):
            samples (list[str]):
            source (str):
            status (ImportFieldRowStatus):
            target (str):
            type_ (str): What ReputeCast stores: email, text, number, date, timezone, language, consent, tags, or – when
                skipped
     """

    label: str
    options: list[ImportFieldOption]
    samples: list[str]
    source: str
    status: ImportFieldRowStatus
    target: str
    type_: str





    def to_dict(self) -> dict[str, Any]:
        from ..models.import_field_option import ImportFieldOption # noqa: PLC0415
        label = self.label

        options = []
        for options_item_data in self.options:
            options_item = options_item_data.to_dict()
            options.append(options_item)



        samples = self.samples



        source = self.source

        status = self.status.value

        target = self.target

        type_ = self.type_


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "label": label,
            "options": options,
            "samples": samples,
            "source": source,
            "status": status,
            "target": target,
            "type": type_,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.import_field_option import ImportFieldOption # noqa: PLC0415
        d = dict(src_dict)
        label = d.pop("label")

        options = []
        _options = d.pop("options")
        for options_item_data in (_options):
            options_item = ImportFieldOption.from_dict(options_item_data)



            options.append(options_item)


        samples = cast(list[str], d.pop("samples"))


        source = d.pop("source")

        status = ImportFieldRowStatus(d.pop("status"))




        target = d.pop("target")

        type_ = d.pop("type")

        import_field_row = cls(
            label=label,
            options=options,
            samples=samples,
            source=source,
            status=status,
            target=target,
            type_=type_,
        )

        return import_field_row

