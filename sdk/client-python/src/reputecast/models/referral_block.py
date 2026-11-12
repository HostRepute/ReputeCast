from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.referral_block_layout import ReferralBlockLayout
from ..types import UNSET, Unset
from typing import cast
from uuid import UUID

if TYPE_CHECKING:
  from ..models.referral_block_texts import ReferralBlockTexts





T = TypeVar("T", bound="ReferralBlock")



@_attrs_define
class ReferralBlock:
    """ 
        Attributes:
            button (str):
            layout (ReferralBlockLayout):
            progress (bool):
            share (list[str]):
            texts (ReferralBlockTexts):
            program (UUID | Unset):
     """

    button: str
    layout: ReferralBlockLayout
    progress: bool
    share: list[str]
    texts: ReferralBlockTexts
    program: UUID | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.referral_block_texts import ReferralBlockTexts # noqa: PLC0415
        button = self.button

        layout = self.layout.value

        progress = self.progress

        share = self.share



        texts = self.texts.to_dict()

        program: str | Unset = UNSET
        if not isinstance(self.program, Unset):
            program = str(self.program)


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "button": button,
            "layout": layout,
            "progress": progress,
            "share": share,
            "texts": texts,
        })
        if program is not UNSET:
            field_dict["program"] = program

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.referral_block_texts import ReferralBlockTexts # noqa: PLC0415
        d = dict(src_dict)
        button = d.pop("button")

        layout = ReferralBlockLayout(d.pop("layout"))




        progress = d.pop("progress")

        share = cast(list[str], d.pop("share"))


        texts = ReferralBlockTexts.from_dict(d.pop("texts"))




        _program = d.pop("program", UNSET)
        program: UUID | Unset
        if isinstance(_program,  Unset):
            program = UNSET
        else:
            program = UUID(_program)




        referral_block = cls(
            button=button,
            layout=layout,
            progress=progress,
            share=share,
            texts=texts,
            program=program,
        )

        return referral_block

