from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.import_consent_status_tone import ImportConsentStatusTone
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.import_field_option import ImportFieldOption





T = TypeVar("T", bound="ImportConsentStatus")



@_attrs_define
class ImportConsentStatus:
    """ 
        Attributes:
            contacts (int):
            effect (str):
            handling (str):
            label (str): The source's own word: subscribed, cleaned, archived with "abuse" complaint
            locked (bool):
            options (list[ImportFieldOption]):
            status (str):
            tone (ImportConsentStatusTone):
            note (str | Unset):
     """

    contacts: int
    effect: str
    handling: str
    label: str
    locked: bool
    options: list[ImportFieldOption]
    status: str
    tone: ImportConsentStatusTone
    note: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.import_field_option import ImportFieldOption # noqa: PLC0415
        contacts = self.contacts

        effect = self.effect

        handling = self.handling

        label = self.label

        locked = self.locked

        options = []
        for options_item_data in self.options:
            options_item = options_item_data.to_dict()
            options.append(options_item)



        status = self.status

        tone = self.tone.value

        note = self.note


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "contacts": contacts,
            "effect": effect,
            "handling": handling,
            "label": label,
            "locked": locked,
            "options": options,
            "status": status,
            "tone": tone,
        })
        if note is not UNSET:
            field_dict["note"] = note

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.import_field_option import ImportFieldOption # noqa: PLC0415
        d = dict(src_dict)
        contacts = d.pop("contacts")

        effect = d.pop("effect")

        handling = d.pop("handling")

        label = d.pop("label")

        locked = d.pop("locked")

        options = []
        _options = d.pop("options")
        for options_item_data in (_options):
            options_item = ImportFieldOption.from_dict(options_item_data)



            options.append(options_item)


        status = d.pop("status")

        tone = ImportConsentStatusTone(d.pop("tone"))




        note = d.pop("note", UNSET)

        import_consent_status = cls(
            contacts=contacts,
            effect=effect,
            handling=handling,
            label=label,
            locked=locked,
            options=options,
            status=status,
            tone=tone,
            note=note,
        )

        return import_consent_status

