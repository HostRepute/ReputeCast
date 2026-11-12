from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="ResumeChanges")



@_attrs_define
class ResumeChanges:
    """ 
        Attributes:
            exclude (int):
            imported_only (bool):
            automation (str | Unset):
            automation_id (str | Unset):
            expected (float | Unset):
            group (str | Unset):
            mailbox (str | Unset):
            rate (float | Unset):
            source (str | Unset):
     """

    exclude: int
    imported_only: bool
    automation: str | Unset = UNSET
    automation_id: str | Unset = UNSET
    expected: float | Unset = UNSET
    group: str | Unset = UNSET
    mailbox: str | Unset = UNSET
    rate: float | Unset = UNSET
    source: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        exclude = self.exclude

        imported_only = self.imported_only

        automation = self.automation

        automation_id = self.automation_id

        expected = self.expected

        group = self.group

        mailbox = self.mailbox

        rate = self.rate

        source = self.source


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "exclude": exclude,
            "importedOnly": imported_only,
        })
        if automation is not UNSET:
            field_dict["automation"] = automation
        if automation_id is not UNSET:
            field_dict["automationId"] = automation_id
        if expected is not UNSET:
            field_dict["expected"] = expected
        if group is not UNSET:
            field_dict["group"] = group
        if mailbox is not UNSET:
            field_dict["mailbox"] = mailbox
        if rate is not UNSET:
            field_dict["rate"] = rate
        if source is not UNSET:
            field_dict["source"] = source

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        exclude = d.pop("exclude")

        imported_only = d.pop("importedOnly")

        automation = d.pop("automation", UNSET)

        automation_id = d.pop("automationId", UNSET)

        expected = d.pop("expected", UNSET)

        group = d.pop("group", UNSET)

        mailbox = d.pop("mailbox", UNSET)

        rate = d.pop("rate", UNSET)

        source = d.pop("source", UNSET)

        resume_changes = cls(
            exclude=exclude,
            imported_only=imported_only,
            automation=automation,
            automation_id=automation_id,
            expected=expected,
            group=group,
            mailbox=mailbox,
            rate=rate,
            source=source,
        )

        return resume_changes

