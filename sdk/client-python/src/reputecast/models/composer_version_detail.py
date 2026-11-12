from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.composer_version_detail_source import ComposerVersionDetailSource
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.composer_ab_test import ComposerABTest





T = TypeVar("T", bound="ComposerVersionDetail")



@_attrs_define
class ComposerVersionDetail:
    """ 
        Attributes:
            current (bool):
            html (str):
            label (str):
            mjml (str):
            name (str):
            preheader (str):
            saved_at (datetime.datetime):
            saved_by (str):
            sections (list[str]):
            size_bytes (int):
            source (ComposerVersionDetailSource):
            subject (str):
            version (int):
            ab_test (ComposerABTest | Unset):
     """

    current: bool
    html: str
    label: str
    mjml: str
    name: str
    preheader: str
    saved_at: datetime.datetime
    saved_by: str
    sections: list[str]
    size_bytes: int
    source: ComposerVersionDetailSource
    subject: str
    version: int
    ab_test: ComposerABTest | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.composer_ab_test import ComposerABTest # noqa: PLC0415
        current = self.current

        html = self.html

        label = self.label

        mjml = self.mjml

        name = self.name

        preheader = self.preheader

        saved_at = self.saved_at.isoformat()

        saved_by = self.saved_by

        sections = self.sections



        size_bytes = self.size_bytes

        source = self.source.value

        subject = self.subject

        version = self.version

        ab_test: dict[str, Any] | Unset = UNSET
        if not isinstance(self.ab_test, Unset):
            ab_test = self.ab_test.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "current": current,
            "html": html,
            "label": label,
            "mjml": mjml,
            "name": name,
            "preheader": preheader,
            "savedAt": saved_at,
            "savedBy": saved_by,
            "sections": sections,
            "sizeBytes": size_bytes,
            "source": source,
            "subject": subject,
            "version": version,
        })
        if ab_test is not UNSET:
            field_dict["abTest"] = ab_test

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.composer_ab_test import ComposerABTest # noqa: PLC0415
        d = dict(src_dict)
        current = d.pop("current")

        html = d.pop("html")

        label = d.pop("label")

        mjml = d.pop("mjml")

        name = d.pop("name")

        preheader = d.pop("preheader")

        saved_at = datetime.datetime.fromisoformat(d.pop("savedAt"))




        saved_by = d.pop("savedBy")

        sections = cast(list[str], d.pop("sections"))


        size_bytes = d.pop("sizeBytes")

        source = ComposerVersionDetailSource(d.pop("source"))




        subject = d.pop("subject")

        version = d.pop("version")

        _ab_test = d.pop("abTest", UNSET)
        ab_test: ComposerABTest | Unset
        if isinstance(_ab_test,  Unset):
            ab_test = UNSET
        else:
            ab_test = ComposerABTest.from_dict(_ab_test)




        composer_version_detail = cls(
            current=current,
            html=html,
            label=label,
            mjml=mjml,
            name=name,
            preheader=preheader,
            saved_at=saved_at,
            saved_by=saved_by,
            sections=sections,
            size_bytes=size_bytes,
            source=source,
            subject=subject,
            version=version,
            ab_test=ab_test,
        )

        return composer_version_detail

