import { useEffect, useState } from 'react'
import { Tree, Button, Modal, Form, Input, message } from 'antd'
import type { DataNode } from 'antd/es/tree'
import { getDepartmentList, createDepartment, Department } from '../../api/department'

const DepartmentList = () => {
  const [data, setData] = useState<Department[]>([])
  const [modalVisible, setModalVisible] = useState(false)
  const [form] = Form.useForm()

  const fetchData = async () => {
    try {
      const resp = await getDepartmentList()
      setData(resp || [])
    } catch (error) {
      console.error('获取部门列表失败:', error)
    }
  }

  useEffect(() => {
    fetchData()
  }, [])

  const buildTreeData = (list: Department[]): DataNode[] => {
    return list.map((item) => ({
      title: item.name,
      key: item.id,
      children: [],
    }))
  }

  const handleAdd = () => {
    form.resetFields()
    setModalVisible(true)
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      await createDepartment(values)
      message.success('创建成功')
      setModalVisible(false)
      fetchData()
    } catch (error) {
      console.error('创建部门失败:', error)
    }
  }

  return (
    <div>
      <h1>部门管理</h1>
      <Button type="primary" onClick={handleAdd} style={{ marginBottom: 16 }}>
        新建部门
      </Button>
      <Tree treeData={buildTreeData(data)} />

      <Modal
        title="新建部门"
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={() => setModalVisible(false)}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="部门名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="parentID" label="上级部门ID">
            <Input type="number" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default DepartmentList