from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.composer_dismissal import ComposerDismissal





T = TypeVar("T", bound="ComposerDismissedIssue")



@_attrs_define
class ComposerDismissedIssue:
    """ 
        Attributes:
            dismissal (ComposerDismissal):
            index (int):
     """

    dismissal: ComposerDismissal
    index: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.composer_dismissal import ComposerDismissal # noqa: PLC0415
        dismissal = self.dismissal.to_dict()

        index = self.index


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "dismissal": dismissal,
            "index": index,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.composer_dismissal import ComposerDismissal # noqa: PLC0415
        d = dict(src_dict)
        dismissal = ComposerDismissal.from_dict(d.pop("dismissal"))




        index = d.pop("index")

        composer_dismissed_issue = cls(
            dismissal=dismissal,
            index=index,
        )

        return composer_dismissed_issue

