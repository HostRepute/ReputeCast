from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.email_language_status import EmailLanguageStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.block_change import BlockChange





T = TypeVar("T", bound="EmailLanguage")



@_attrs_define
class EmailLanguage:
    """ 
        Attributes:
            attention (int):
            blocks (int):
            changed (int):
            changed_blocks (list[str]):
            changed_indexes (list[int]):
            gets (str):
            held (bool):
            language (str):
            name (str):
            native (str):
            no_language (int):
            not_offered (int):
            offered (bool):
            preheader (str):
            preheader_changed (bool):
            status (EmailLanguageStatus):
            still (int):
            still_indexes (list[int]):
            subject (str):
            subject_changed (bool):
            translated (int):
            translated_by (str):
            translated_from (int):
            up_to_date (int):
            updated_by (str):
            version (int):
            last_change (BlockChange | Unset):
            readers (int | Unset):
            translated_at (datetime.datetime | Unset):
            updated_at (datetime.datetime | Unset):
     """

    attention: int
    blocks: int
    changed: int
    changed_blocks: list[str]
    changed_indexes: list[int]
    gets: str
    held: bool
    language: str
    name: str
    native: str
    no_language: int
    not_offered: int
    offered: bool
    preheader: str
    preheader_changed: bool
    status: EmailLanguageStatus
    still: int
    still_indexes: list[int]
    subject: str
    subject_changed: bool
    translated: int
    translated_by: str
    translated_from: int
    up_to_date: int
    updated_by: str
    version: int
    last_change: BlockChange | Unset = UNSET
    readers: int | Unset = UNSET
    translated_at: datetime.datetime | Unset = UNSET
    updated_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.block_change import BlockChange # noqa: PLC0415
        attention = self.attention

        blocks = self.blocks

        changed = self.changed

        changed_blocks = self.changed_blocks



        changed_indexes = self.changed_indexes



        gets = self.gets

        held = self.held

        language = self.language

        name = self.name

        native = self.native

        no_language = self.no_language

        not_offered = self.not_offered

        offered = self.offered

        preheader = self.preheader

        preheader_changed = self.preheader_changed

        status = self.status.value

        still = self.still

        still_indexes = self.still_indexes



        subject = self.subject

        subject_changed = self.subject_changed

        translated = self.translated

        translated_by = self.translated_by

        translated_from = self.translated_from

        up_to_date = self.up_to_date

        updated_by = self.updated_by

        version = self.version

        last_change: dict[str, Any] | Unset = UNSET
        if not isinstance(self.last_change, Unset):
            last_change = self.last_change.to_dict()

        readers = self.readers

        translated_at: str | Unset = UNSET
        if not isinstance(self.translated_at, Unset):
            translated_at = self.translated_at.isoformat()

        updated_at: str | Unset = UNSET
        if not isinstance(self.updated_at, Unset):
            updated_at = self.updated_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "attention": attention,
            "blocks": blocks,
            "changed": changed,
            "changedBlocks": changed_blocks,
            "changedIndexes": changed_indexes,
            "gets": gets,
            "held": held,
            "language": language,
            "name": name,
            "native": native,
            "noLanguage": no_language,
            "notOffered": not_offered,
            "offered": offered,
            "preheader": preheader,
            "preheaderChanged": preheader_changed,
            "status": status,
            "still": still,
            "stillIndexes": still_indexes,
            "subject": subject,
            "subjectChanged": subject_changed,
            "translated": translated,
            "translatedBy": translated_by,
            "translatedFrom": translated_from,
            "upToDate": up_to_date,
            "updatedBy": updated_by,
            "version": version,
        })
        if last_change is not UNSET:
            field_dict["lastChange"] = last_change
        if readers is not UNSET:
            field_dict["readers"] = readers
        if translated_at is not UNSET:
            field_dict["translatedAt"] = translated_at
        if updated_at is not UNSET:
            field_dict["updatedAt"] = updated_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.block_change import BlockChange # noqa: PLC0415
        d = dict(src_dict)
        attention = d.pop("attention")

        blocks = d.pop("blocks")

        changed = d.pop("changed")

        changed_blocks = cast(list[str], d.pop("changedBlocks"))


        changed_indexes = cast(list[int], d.pop("changedIndexes"))


        gets = d.pop("gets")

        held = d.pop("held")

        language = d.pop("language")

        name = d.pop("name")

        native = d.pop("native")

        no_language = d.pop("noLanguage")

        not_offered = d.pop("notOffered")

        offered = d.pop("offered")

        preheader = d.pop("preheader")

        preheader_changed = d.pop("preheaderChanged")

        status = EmailLanguageStatus(d.pop("status"))




        still = d.pop("still")

        still_indexes = cast(list[int], d.pop("stillIndexes"))


        subject = d.pop("subject")

        subject_changed = d.pop("subjectChanged")

        translated = d.pop("translated")

        translated_by = d.pop("translatedBy")

        translated_from = d.pop("translatedFrom")

        up_to_date = d.pop("upToDate")

        updated_by = d.pop("updatedBy")

        version = d.pop("version")

        _last_change = d.pop("lastChange", UNSET)
        last_change: BlockChange | Unset
        if isinstance(_last_change,  Unset):
            last_change = UNSET
        else:
            last_change = BlockChange.from_dict(_last_change)




        readers = d.pop("readers", UNSET)

        _translated_at = d.pop("translatedAt", UNSET)
        translated_at: datetime.datetime | Unset
        if isinstance(_translated_at,  Unset):
            translated_at = UNSET
        else:
            translated_at = datetime.datetime.fromisoformat(_translated_at)




        _updated_at = d.pop("updatedAt", UNSET)
        updated_at: datetime.datetime | Unset
        if isinstance(_updated_at,  Unset):
            updated_at = UNSET
        else:
            updated_at = datetime.datetime.fromisoformat(_updated_at)




        email_language = cls(
            attention=attention,
            blocks=blocks,
            changed=changed,
            changed_blocks=changed_blocks,
            changed_indexes=changed_indexes,
            gets=gets,
            held=held,
            language=language,
            name=name,
            native=native,
            no_language=no_language,
            not_offered=not_offered,
            offered=offered,
            preheader=preheader,
            preheader_changed=preheader_changed,
            status=status,
            still=still,
            still_indexes=still_indexes,
            subject=subject,
            subject_changed=subject_changed,
            translated=translated,
            translated_by=translated_by,
            translated_from=translated_from,
            up_to_date=up_to_date,
            updated_by=updated_by,
            version=version,
            last_change=last_change,
            readers=readers,
            translated_at=translated_at,
            updated_at=updated_at,
        )

        return email_language

