from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.transactional_template_create_editor_mode import TransactionalTemplateCreateEditorMode
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.transactional_template_create_origin import TransactionalTemplateCreateOrigin
  from ..models.transactional_test_data import TransactionalTestData
  from ..models.transactional_variable import TransactionalVariable





T = TypeVar("T", bound="TransactionalTemplateCreate")



@_attrs_define
class TransactionalTemplateCreate:
    """ 
        Attributes:
            name (str):
            category_id (str | Unset): Defaults to the first Essential category
            description (str | Unset):
            editor_mode (TransactionalTemplateCreateEditorMode | Unset):
            from_template (str | Unset):
            html (str | Unset):
            key (str | Unset): Defaults to the name in lowercase with hyphens
            mjml (str | Unset):
            origin (TransactionalTemplateCreateOrigin | Unset):
            plain_text (str | Unset):
            preheader (str | Unset):
            starter (str | Unset): A starter's key (password-reset, magic-link…)
            subject (str | Unset):
            test_data (list[TransactionalTestData] | Unset):
            variables (list[TransactionalVariable] | Unset):
     """

    name: str
    category_id: str | Unset = UNSET
    description: str | Unset = UNSET
    editor_mode: TransactionalTemplateCreateEditorMode | Unset = UNSET
    from_template: str | Unset = UNSET
    html: str | Unset = UNSET
    key: str | Unset = UNSET
    mjml: str | Unset = UNSET
    origin: TransactionalTemplateCreateOrigin | Unset = UNSET
    plain_text: str | Unset = UNSET
    preheader: str | Unset = UNSET
    starter: str | Unset = UNSET
    subject: str | Unset = UNSET
    test_data: list[TransactionalTestData] | Unset = UNSET
    variables: list[TransactionalVariable] | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.transactional_template_create_origin import TransactionalTemplateCreateOrigin # noqa: PLC0415
        from ..models.transactional_test_data import TransactionalTestData # noqa: PLC0415
        from ..models.transactional_variable import TransactionalVariable # noqa: PLC0415
        name = self.name

        category_id = self.category_id

        description = self.description

        editor_mode: str | Unset = UNSET
        if not isinstance(self.editor_mode, Unset):
            editor_mode = self.editor_mode.value


        from_template = self.from_template

        html = self.html

        key = self.key

        mjml = self.mjml

        origin: dict[str, Any] | Unset = UNSET
        if not isinstance(self.origin, Unset):
            origin = self.origin.to_dict()

        plain_text = self.plain_text

        preheader = self.preheader

        starter = self.starter

        subject = self.subject

        test_data: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.test_data, Unset):
            test_data = []
            for test_data_item_data in self.test_data:
                test_data_item = test_data_item_data.to_dict()
                test_data.append(test_data_item)



        variables: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.variables, Unset):
            variables = []
            for variables_item_data in self.variables:
                variables_item = variables_item_data.to_dict()
                variables.append(variables_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "name": name,
        })
        if category_id is not UNSET:
            field_dict["categoryId"] = category_id
        if description is not UNSET:
            field_dict["description"] = description
        if editor_mode is not UNSET:
            field_dict["editorMode"] = editor_mode
        if from_template is not UNSET:
            field_dict["fromTemplate"] = from_template
        if html is not UNSET:
            field_dict["html"] = html
        if key is not UNSET:
            field_dict["key"] = key
        if mjml is not UNSET:
            field_dict["mjml"] = mjml
        if origin is not UNSET:
            field_dict["origin"] = origin
        if plain_text is not UNSET:
            field_dict["plainText"] = plain_text
        if preheader is not UNSET:
            field_dict["preheader"] = preheader
        if starter is not UNSET:
            field_dict["starter"] = starter
        if subject is not UNSET:
            field_dict["subject"] = subject
        if test_data is not UNSET:
            field_dict["testData"] = test_data
        if variables is not UNSET:
            field_dict["variables"] = variables

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transactional_template_create_origin import TransactionalTemplateCreateOrigin # noqa: PLC0415
        from ..models.transactional_test_data import TransactionalTestData # noqa: PLC0415
        from ..models.transactional_variable import TransactionalVariable # noqa: PLC0415
        d = dict(src_dict)
        name = d.pop("name")

        category_id = d.pop("categoryId", UNSET)

        description = d.pop("description", UNSET)

        _editor_mode = d.pop("editorMode", UNSET)
        editor_mode: TransactionalTemplateCreateEditorMode | Unset
        if isinstance(_editor_mode,  Unset):
            editor_mode = UNSET
        else:
            editor_mode = TransactionalTemplateCreateEditorMode(_editor_mode)




        from_template = d.pop("fromTemplate", UNSET)

        html = d.pop("html", UNSET)

        key = d.pop("key", UNSET)

        mjml = d.pop("mjml", UNSET)

        _origin = d.pop("origin", UNSET)
        origin: TransactionalTemplateCreateOrigin | Unset
        if isinstance(_origin,  Unset):
            origin = UNSET
        else:
            origin = TransactionalTemplateCreateOrigin.from_dict(_origin)




        plain_text = d.pop("plainText", UNSET)

        preheader = d.pop("preheader", UNSET)

        starter = d.pop("starter", UNSET)

        subject = d.pop("subject", UNSET)

        _test_data = d.pop("testData", UNSET)
        test_data: list[TransactionalTestData] | Unset = UNSET
        if _test_data is not UNSET:
            test_data = []
            for test_data_item_data in _test_data:
                test_data_item = TransactionalTestData.from_dict(test_data_item_data)



                test_data.append(test_data_item)


        _variables = d.pop("variables", UNSET)
        variables: list[TransactionalVariable] | Unset = UNSET
        if _variables is not UNSET:
            variables = []
            for variables_item_data in _variables:
                variables_item = TransactionalVariable.from_dict(variables_item_data)



                variables.append(variables_item)


        transactional_template_create = cls(
            name=name,
            category_id=category_id,
            description=description,
            editor_mode=editor_mode,
            from_template=from_template,
            html=html,
            key=key,
            mjml=mjml,
            origin=origin,
            plain_text=plain_text,
            preheader=preheader,
            starter=starter,
            subject=subject,
            test_data=test_data,
            variables=variables,
        )

        return transactional_template_create

