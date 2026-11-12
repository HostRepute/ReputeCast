from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.composer_ab_test import ComposerABTest





T = TypeVar("T", bound="ComposerChecksRequest")



@_attrs_define
class ComposerChecksRequest:
    """ 
        Attributes:
            ab_test (ComposerABTest | Unset):
            html (str | Unset):
            language (str | Unset):
            mjml (str | Unset):
            preheader (str | Unset):
            probe_links (bool | Unset): Also request every link (slower; Send Readiness does it)
            subject (str | Unset):
     """

    ab_test: ComposerABTest | Unset = UNSET
    html: str | Unset = UNSET
    language: str | Unset = UNSET
    mjml: str | Unset = UNSET
    preheader: str | Unset = UNSET
    probe_links: bool | Unset = UNSET
    subject: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.composer_ab_test import ComposerABTest # noqa: PLC0415
        ab_test: dict[str, Any] | Unset = UNSET
        if not isinstance(self.ab_test, Unset):
            ab_test = self.ab_test.to_dict()

        html = self.html

        language = self.language

        mjml = self.mjml

        preheader = self.preheader

        probe_links = self.probe_links

        subject = self.subject


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if ab_test is not UNSET:
            field_dict["abTest"] = ab_test
        if html is not UNSET:
            field_dict["html"] = html
        if language is not UNSET:
            field_dict["language"] = language
        if mjml is not UNSET:
            field_dict["mjml"] = mjml
        if preheader is not UNSET:
            field_dict["preheader"] = preheader
        if probe_links is not UNSET:
            field_dict["probeLinks"] = probe_links
        if subject is not UNSET:
            field_dict["subject"] = subject

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.composer_ab_test import ComposerABTest # noqa: PLC0415
        d = dict(src_dict)
        _ab_test = d.pop("abTest", UNSET)
        ab_test: ComposerABTest | Unset
        if isinstance(_ab_test,  Unset):
            ab_test = UNSET
        else:
            ab_test = ComposerABTest.from_dict(_ab_test)




        html = d.pop("html", UNSET)

        language = d.pop("language", UNSET)

        mjml = d.pop("mjml", UNSET)

        preheader = d.pop("preheader", UNSET)

        probe_links = d.pop("probeLinks", UNSET)

        subject = d.pop("subject", UNSET)

        composer_checks_request = cls(
            ab_test=ab_test,
            html=html,
            language=language,
            mjml=mjml,
            preheader=preheader,
            probe_links=probe_links,
            subject=subject,
        )

        return composer_checks_request

