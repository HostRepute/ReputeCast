from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.merged_section import MergedSection





T = TypeVar("T", bound="ComposerMerged")



@_attrs_define
class ComposerMerged:
    """ 
        Attributes:
            sections (list[MergedSection]): Their sections, at their positions in the email
            version (int): Their version
            with_ (str): Who saved the newer version
            head (str | Unset):
            preheader (str | Unset):
            subject (str | Unset):
     """

    sections: list[MergedSection]
    version: int
    with_: str
    head: str | Unset = UNSET
    preheader: str | Unset = UNSET
    subject: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.merged_section import MergedSection # noqa: PLC0415
        sections = []
        for sections_item_data in self.sections:
            sections_item = sections_item_data.to_dict()
            sections.append(sections_item)



        version = self.version

        with_ = self.with_

        head = self.head

        preheader = self.preheader

        subject = self.subject


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "sections": sections,
            "version": version,
            "with": with_,
        })
        if head is not UNSET:
            field_dict["head"] = head
        if preheader is not UNSET:
            field_dict["preheader"] = preheader
        if subject is not UNSET:
            field_dict["subject"] = subject

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.merged_section import MergedSection # noqa: PLC0415
        d = dict(src_dict)
        sections = []
        _sections = d.pop("sections")
        for sections_item_data in (_sections):
            sections_item = MergedSection.from_dict(sections_item_data)



            sections.append(sections_item)


        version = d.pop("version")

        with_ = d.pop("with")

        head = d.pop("head", UNSET)

        preheader = d.pop("preheader", UNSET)

        subject = d.pop("subject", UNSET)

        composer_merged = cls(
            sections=sections,
            version=version,
            with_=with_,
            head=head,
            preheader=preheader,
            subject=subject,
        )

        return composer_merged

