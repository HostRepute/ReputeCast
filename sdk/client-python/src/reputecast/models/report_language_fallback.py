from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.report_language_draft import ReportLanguageDraft





T = TypeVar("T", bound="ReportLanguageFallback")



@_attrs_define
class ReportLanguageFallback:
    """ 
        Attributes:
            got (str):
            readers (list[str]):
            draft (ReportLanguageDraft | Unset):
            gap (float | Unset):
     """

    got: str
    readers: list[str]
    draft: ReportLanguageDraft | Unset = UNSET
    gap: float | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.report_language_draft import ReportLanguageDraft # noqa: PLC0415
        got = self.got

        readers = self.readers



        draft: dict[str, Any] | Unset = UNSET
        if not isinstance(self.draft, Unset):
            draft = self.draft.to_dict()

        gap = self.gap


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "got": got,
            "readers": readers,
        })
        if draft is not UNSET:
            field_dict["draft"] = draft
        if gap is not UNSET:
            field_dict["gap"] = gap

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.report_language_draft import ReportLanguageDraft # noqa: PLC0415
        d = dict(src_dict)
        got = d.pop("got")

        readers = cast(list[str], d.pop("readers"))


        _draft = d.pop("draft", UNSET)
        draft: ReportLanguageDraft | Unset
        if isinstance(_draft,  Unset):
            draft = UNSET
        else:
            draft = ReportLanguageDraft.from_dict(_draft)




        gap = d.pop("gap", UNSET)

        report_language_fallback = cls(
            got=got,
            readers=readers,
            draft=draft,
            gap=gap,
        )

        return report_language_fallback

